package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/echocat/slf4g"
	"github.com/echocat/slf4g/fields"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	gcv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/engity-com/bifroest/pkg/common"
)

func newBuildImage(b *build) *buildImage {
	return &buildImage{
		build: b,

		defaultConfigFile: "contrib/configurations/sshd-dropin-replacement.yaml",
	}
}

type buildImage struct {
	*build

	defaultConfigFile string
}

func (this *buildImage) attach(cmd *kingpin.CmdClause) {
	cmd.Flag("defaultConfigFile", "").
		Default(this.defaultConfigFile).
		PlaceHolder("<file>").
		StringVar(&this.defaultConfigFile)
}

func (this *buildImage) create(ctx context.Context, binary *buildArtifact) (_ buildArtifacts, rErr error) {
	var result buildArtifacts

	success := false
	a, err := this.createPart(ctx, binary)
	if err != nil {
		return nil, err
	}
	defer common.IgnoreCloseErrorIfFalse(&success, a)
	result = append(result, a)

	success = true
	return result, nil
}

func (this *buildImage) createPart(ctx context.Context, binary *buildArtifact) (_ *buildArtifact, rErr error) {
	success := false
	a, err := this.build.newBuildArtifact(ctx, binary.platform, buildArtifactTypeImage)
	if err != nil {
		return nil, err
	}
	defer common.IgnoreCloseErrorIfFalse(&success, a)

	fail := func(err error) (*buildArtifact, error) {
		return nil, fmt.Errorf("cannot create %v: %w", a, err)
	}

	var from string
	if from, err = a.from(); err != nil {
		return fail(err)
	}

	l := log.With("platform", a.platform).
		With("from", from).
		With("stage", buildStageImage)

	start := time.Now()
	l.Debug("building image...")

	ociPlatform, err := gcv1.ParsePlatform(a.ociString())
	if err != nil {
		return fail(err)
	}

	var img gcv1.Image
	if strings.EqualFold(from, "scratch") {
		img = empty.Image
		img = mutate.MediaType(img, types.OCIManifestSchema1)
		img = mutate.ConfigMediaType(img, types.OCIConfigJSON)
	} else {
		if img, err = crane.Pull(from,
			crane.WithPlatform(ociPlatform),
			crane.WithContext(ctx),
		); err != nil {
			return fail(err)
		}
	}

	deps, err := this.dependencies.imagesFiles.downloadFilesFor(ctx, a.os, a.arch)
	if err != nil {
		return fail(err)
	}
	artifactDepItems := make([]imageArtifactLayerItem, len(deps))
	for i, dep := range deps {
		artifactDepItems[i] = imageArtifactLayerItem{
			sourceFile: dep.source,
			targetFile: dep.target,
			mode:       dep.mode,
		}
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fail(err)
	}
	cfg = cfg.DeepCopy()
	cfg.Architecture = ociPlatform.Architecture
	cfg.OS = ociPlatform.OS
	cfg.OSVersion = ociPlatform.OSVersion
	cfg.OSFeatures = ociPlatform.OSFeatures
	cfg.Variant = ociPlatform.Variant

	cfg.Config.Labels = map[string]string{
		v1.AnnotationDescription: "Test test",
		v1.AnnotationSource:      this.repo.fullName(),
	}
	cfg.Config.Env = binary.os.extendPathWith(binary.platform.os.bifroestBinaryDirPath(), cfg.Config.Env)
	cfg.Config.Entrypoint = []string{binary.platform.os.bifroestBinaryFilePath()}
	cfg.Config.Cmd = []string{"run"}
	cfg.Config.ExposedPorts = map[string]struct{}{
		"22/tcp": {},
	}

	img, err = mutate.ConfigFile(img, cfg)
	if err != nil {
		return fail(err)
	}

	img = mutate.Annotations(img, map[string]string{
		v1.AnnotationDescription: "This is a description " + a.platform.String(),
	}).(gcv1.Image)

	binaryLayer, err := binary.toLayer(common.JoinSeq2[imageArtifactLayerItem, error](
		common.Seq2ErrOf[imageArtifactLayerItem](imageArtifactLayerItem{
			sourceFile: this.defaultConfigFile,
			targetFile: binary.platform.os.bifroestConfigFilePath(),
			mode:       0644,
		}),
		common.Seq2ErrOf[imageArtifactLayerItem](artifactDepItems...),
	))
	if err != nil {
		return fail(err)
	}

	if img, err = mutate.AppendLayers(img, binaryLayer); err != nil {
		return fail(err)
	}

	a.ociImage = img

	ld := l.With("duration", time.Since(start).Truncate(time.Millisecond))
	if l.IsDebugEnabled() {
		ld.Debug("building image... DONE!")
	} else {
		ld.Info("image built")
	}

	success = true
	return a, nil
}

func (this *buildImage) merge(_ context.Context, as buildArtifacts) (_ buildArtifacts, rErr error) {
	result := slices.Collect(as.withoutType(buildArtifactTypeImage))

	success := false
	for _, e := range allEditionVariants {
		a, err := this.createdMerged(e, as)
		if err != nil {
			return nil, err
		}
		defer common.IgnoreCloseErrorIfFalse(&success, a)
		if a != nil {
			result = append(result, a)
		}
	}

	success = true
	return result, nil
}

func (this *buildImage) createdMerged(e edition, as buildArtifacts) (result *buildArtifact, _ error) {
	l := log.With("edition", e).
		With("stage", buildStageImage)

	start := time.Now()
	l.Debug("merge images...")

	var manifest gcv1.ImageIndex = empty.Index
	mutate.IndexMediaType(manifest, types.DockerManifestList)
	manifest = mutate.Annotations(manifest, map[string]string{
		v1.AnnotationSource:      this.repo.fullName(),
		v1.AnnotationDescription: "This is a description",
	}).(gcv1.ImageIndex)

	var adds []mutate.IndexAddendum
	var refA *buildArtifact

	for aa := range as.filter(func(candidate *buildArtifact) bool {
		return candidate.edition == e && candidate.t == buildArtifactTypeImage
	}) {
		fail := func(err error) (*buildArtifact, error) {
			return nil, fmt.Errorf("cannot merge artifact %v: %w", aa, err)
		}

		cf, err := aa.ociImage.ConfigFile()
		if err != nil {
			return fail(err)
		}

		newDesc, err := partial.Descriptor(aa.ociImage)
		if err != nil {
			return fail(err)
		}
		newDesc.Platform = cf.Platform()
		adds = append(adds, mutate.IndexAddendum{
			Add:        aa.ociImage,
			Descriptor: *newDesc,
		})
		refA = aa
	}

	success := false
	if refA != nil {
		result = &buildArtifact{
			platform: &platform{
				edition: e,
				testing: refA.testing,
			},
			buildContext: refA.buildContext,
			t:            buildArtifactTypeImagePlatform,
			ociIndex:     mutate.AppendManifests(manifest, adds...),
		}
		defer common.IgnoreCloseErrorIfFalse(&success, result)
	}

	ld := l.With("duration", time.Since(start).Truncate(time.Millisecond))
	if l.IsDebugEnabled() {
		if result != nil {
			ld.Debug("merge images... DONE!")
		} else {
			ld.Debug("merge images... SKIPPED! (none found)")
		}
	} else if result != nil {
		ld.Info("images merged")
	}
	success = true
	return result, nil
}

func (this *buildImage) publish(ctx context.Context, as buildArtifacts) (rErr error) {
	fail := func(err error) error {
		return fmt.Errorf("cannot publish artifacts: %w", err)
	}

	for a := range as.onlyOfType(buildArtifactTypeImagePlatform) {
		l := log.With("edition", a.edition).
			With("stage", buildStagePublish)

		start := time.Now()
		l.Debug("push images...")

		refs, err := this.refs(ctx, a.edition)
		if err != nil {
			return fail(err)
		}
		l = l.With("refs", this.lazyRefs(&refs))

		for _, ref := range refs {
			if err := remote.WriteIndex(ref, a.ociIndex,
				remote.WithContext(ctx),
				remote.WithAuth(&authn.Basic{
					Username: this.actor,
					Password: this.repo.githubToken,
				}),
			); err != nil {
				return fail(err)
			}
		}

		ld := l.With("duration", time.Since(start).Truncate(time.Millisecond))
		if l.IsDebugEnabled() {
			ld.Debug("push images... DONE!")
		} else {
			ld.Info("images pushed")
		}
	}

	return nil
}

func (this *buildImage) refs(ctx context.Context, e edition) ([]name.Reference, error) {
	v, err := this.version(ctx)
	if err != nil {
		return nil, err
	}

	var rs []name.Reference
	prefix := e.String() + "-"
	root := e.String()
	for tag := range v.tags(prefix, root) {
		r, err := name.ParseReference(this.repo.fullImageName() + ":" + tag)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}

	if e == editionGeneric {
		for tag := range v.tags("", "latest") {
			r, err := name.ParseReference(this.repo.fullImageName() + ":" + tag)
			if err != nil {
				return nil, err
			}
			rs = append(rs, r)
		}
	}

	return rs, nil
}

func (this *buildImage) lazyRefs(p *[]name.Reference) fields.Lazy {
	return fields.LazyFunc(func() any {
		if p == nil || len(*p) == 0 {
			return fields.Exclude
		}
		result := make([]string, len(*p))
		for i, r := range *p {
			result[i] = r.String()
		}
		if len(result) == 1 {
			return result[0]
		}
		return result
	})
}
