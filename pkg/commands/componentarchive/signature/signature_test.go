package signature_test

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	"github.com/gardener/component-spec/bindings-go/ctf"

	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/signature/verify"
	"github.com/gardener/component-cli/pkg/signatures"
	"github.com/gardener/component-cli/pkg/testutils"
	. "github.com/onsi/ginkgo"
)

func getParentCd() cdv2.ComponentDescriptor {
	refResParent := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0/test-resource-parent:v0.0.1")
	uploadTestResource(refResParent, "data-parent")
	parentResAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refResParent))
	Expect(err).ToNot(HaveOccurred())
	return cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			ObjectMeta: cdv2.ObjectMeta{
				Name:    "github.com/component-cli/test-component-parent",
				Version: "v0.1.0",
			},
			Provider: cdv2.InternalProvider,
			ComponentReferences: []v2.ComponentReference{
				{
					Name:          "test-component-child",
					ComponentName: "github.com/component-cli/test-component-child",
					Version:       "v0.1.0",
					ExtraIdentity: v2.Identity{
						"refkey": "refName",
					},
				},
			},
			Resources: []cdv2.Resource{
				{
					IdentityObjectMeta: cdv2.IdentityObjectMeta{
						Name:    "resource1",
						Version: "v0.0.1",
						ExtraIdentity: cdv2.Identity{
							"key": "value",
						},
						Type: "ociImage",
					},
					Access: &parentResAccess,
				},
			},
		},
	}
}

func getChildCd() cdv2.ComponentDescriptor {
	refResChild := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0/test-resource-child:v0.0.1")
	uploadTestResource(refResChild, "data-child")
	childResAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refResChild))
	Expect(err).ToNot(HaveOccurred())
	return cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			ObjectMeta: cdv2.ObjectMeta{
				Name:    "github.com/component-cli/test-component-child",
				Version: "v0.1.0",
			},
			Provider: cdv2.InternalProvider,
			Resources: []cdv2.Resource{
				{
					IdentityObjectMeta: cdv2.IdentityObjectMeta{
						Name:    "resource2",
						Version: "v0.0.1",
						ExtraIdentity: cdv2.Identity{
							"key": "value2",
						},
						Type: "ociImage",
					},
					Access: &childResAccess,
				},
			},
		},
	}
}

func uploadTestResource(ref string, layerData string) {
	ctx := context.Background()
	defer ctx.Done()

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-1-data"),
		[]byte(layerData),
	}

	testutils.UploadTestImage(ctx, client, ref, ocispecv1.MediaTypeImageManifest, configData, layersData)
}

func uploadTestCd(cd cdv2.ComponentDescriptor, ref string) {
	ctx := context.TODO()
	fs := memoryfs.New()

	ociRepo := cdv2.NewOCIRegistryRepository(ref, "")
	repoCtx, err := cdv2.NewUnstructured(
		ociRepo,
	)
	Expect(err).ToNot(HaveOccurred())

	err = cdv2.InjectRepositoryContext(&cd, &repoCtx)
	Expect(err).ToNot(HaveOccurred())

	manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(&cd, fs)).Build(ctx)
	Expect(err).ToNot(HaveOccurred())

	ociRef, err := cdoci.OCIRef(*ociRepo, cd.Name, cd.Version)
	Expect(err).ToNot(HaveOccurred())

	Expect(client.PushManifest(ctx, ociRef, manifest)).To(Succeed())

	cdresolver := cdoci.NewResolver(client)
	_, err = cdresolver.Resolve(ctx, ociRepo, cd.Name, cd.Version)
	Expect(err).ToNot(HaveOccurred())

}

var _ = Describe("signature", func() {
	Context("add digest", func() {
		It("should add digests to a cd and referenced cd", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedChildCd := digestedCds[0]
			digestedParentCd := digestedCds[1]

			Expect(digestedChildCd.Resources[0].Digest).ToNot(BeNil())
			Expect(digestedChildCd.Resources[0].Digest).ToNot(Equal(cdv2.NewExcludeFromSignatureDigest()))

			Expect(digestedParentCd.Resources[0].Digest).ToNot(BeNil())
			Expect(digestedParentCd.Resources[0].Digest).ToNot(Equal(cdv2.NewExcludeFromSignatureDigest()))

			Expect(digestedParentCd.ComponentReferences[0].Digest).ToNot(BeNil())
			Expect(digestedParentCd.ComponentReferences[0].Digest).ToNot(Equal(cdv2.NewExcludeFromSignatureDigest()))
		})

		It("should fail to add digests if preexisting digest in component ref mismatches calculated digest", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			//add a wrong digest
			parentCd.ComponentReferences[0].Digest = &cdv2.DigestSpec{
				HashAlgorithm:          "FAKE",
				NormalisationAlgorithm: "FAKE",
				Value:                  "FAKE",
			}

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).To(HaveOccurred())
			Expect(digestedCds).To(BeNil())
		})
		It("should fail to add digests if preexisting digest in parent cd resource mismatches calculated digest", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			//add a wrong digest
			parentCd.Resources[0].Digest = &cdv2.DigestSpec{
				HashAlgorithm:          "FAKE",
				NormalisationAlgorithm: "FAKE",
				Value:                  "FAKE",
			}

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).To(HaveOccurred())
			Expect(digestedCds).To(BeNil())
		})
		It("should fail to add digests if preexisting digest in child cd resource mismatches calculated digest", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			//add a wrong digest
			childCd.Resources[0].Digest = &cdv2.DigestSpec{
				HashAlgorithm:          "FAKE",
				NormalisationAlgorithm: "FAKE",
				Value:                  "FAKE",
			}

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).To(HaveOccurred())
			Expect(digestedCds).To(BeNil())
		})
		It("should add a exclude-from-signature digest to skip-access-types", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{"ociRegistry": true})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedChildCd := digestedCds[0]
			digestedParentCd := digestedCds[1]

			Expect(digestedChildCd.Resources[0].Digest).ToNot(BeNil())
			Expect(digestedChildCd.Resources[0].Digest).To(Equal(cdv2.NewExcludeFromSignatureDigest()))

			Expect(digestedParentCd.Resources[0].Digest).ToNot(BeNil())
			Expect(digestedParentCd.Resources[0].Digest).To(Equal(cdv2.NewExcludeFromSignatureDigest()))

			Expect(digestedParentCd.ComponentReferences[0].Digest).ToNot(BeNil())
			Expect(digestedParentCd.ComponentReferences[0].Digest).ToNot(Equal(cdv2.NewExcludeFromSignatureDigest()))
		})

	})

	Context("verify", func() {
		It("should verify a cd with referenced cd and resource each", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should succeed with resource in different location", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			//change location of resource of child
			refResChild := fmt.Sprintf("%s/%s", testenv.Addr, "different-location/test-resource-child:v0.0.1")
			uploadTestResource(refResChild, "data-child")
			childResAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refResChild))
			Expect(err).ToNot(HaveOccurred())
			childCd.Resources[0].Access = &childResAccess
			uploadTestCd(childCd, ref)

			//change location of resource of parent
			refResParent := fmt.Sprintf("%s/%s", testenv.Addr, "different-location/test-resource-parent:v0.0.1")
			uploadTestResource(refResParent, "data-parent")
			parentResAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refResParent))
			Expect(err).ToNot(HaveOccurred())
			parentCd.Resources[0].Access = &parentResAccess
			uploadTestCd(parentCd, ref)

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail verify with manipulated resource in child", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			//change access imageReference of child to wrong resource (=resource of parent) --> content=digest is different
			refWrongResource := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0/test-resource-parent:v0.0.1")
			wrongResourceAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refWrongResource))
			Expect(err).ToNot(HaveOccurred())
			childCd.Resources[0].Access = &wrongResourceAccess
			uploadTestCd(childCd, ref)

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())
		})
		It("should fail verify with manipulated resource in parent", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			//change access imageReference of parent to wrong resource (= resource of child) --> content=digest is different
			refWrongResource := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0/test-resource-child:v0.0.1")
			wrongResourceAccess, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(refWrongResource))
			Expect(err).ToNot(HaveOccurred())
			digestedParentCd.Resources[0].Access = &wrongResourceAccess

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())
		})
		It("should fail verify with component reference digest manipulation", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			//manipulate digest of component descriptor
			digestedParentCd.ComponentReferences[0].Digest.Value = "faked"

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())
		})
		It("should fail verify with access type manipulation", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1]

			//set resource access nil -> content will not be digested
			digestedParentCd.Resources[0].Access = nil

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())

			//set resource access none -> content will not be digested
			digestedParentCd.Resources[0].Access = cdv2.NewEmptyUnstructured("None")

			err = verify.CheckCdDigests(digestedParentCd, *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())
		})
		It("should fail verify with exclude-from-signature manipulation", func() {
			parentCd := getParentCd()
			childCd := getChildCd()

			ref := fmt.Sprintf("%s/%s", testenv.Addr, "cd/0")
			uploadTestCd(parentCd, ref)
			uploadTestCd(childCd, ref)

			//add digests
			digestedCds, err := signatures.RecursivelyAddDigestsToCd(&parentCd, *cdv2.NewOCIRegistryRepository(ref, ""), client, map[string]ctf.BlobResolver{}, context.TODO(), map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(digestedCds)).To(Equal(2))

			digestedParentCd := digestedCds[1].DeepCopy()

			//calculate hash
			hasher, err := cdv2Sign.HasherForName("sha256")
			Expect(err).ToNot(HaveOccurred())

			hashedDigestOriginal, err := cdv2Sign.HashForComponentDescriptor(*digestedParentCd, *hasher)
			Expect(err).ToNot(HaveOccurred())

			//set parent resource to exclude from digest
			digestedParentCd.Resources[0].Digest = cdv2.NewExcludeFromSignatureDigest()

			//check that cd digest is not the same
			hashedDigestExcludeFromSignature, err := cdv2Sign.HashForComponentDescriptor(*digestedParentCd, *hasher)
			Expect(err).ToNot(HaveOccurred())

			Expect(hashedDigestOriginal).ToNot(Equal(hashedDigestExcludeFromSignature))

			//set child resource to exlcude from digest (without manipulating parent)
			childCd.Resources[0].Digest = cdv2.NewExcludeFromSignatureDigest()
			uploadTestCd(childCd, ref)

			repoCtx := cdv2.NewOCIRegistryRepository(ref, "")
			err = verify.CheckCdDigests(digestedCds[1], *repoCtx, client, context.TODO())
			Expect(err).To(HaveOccurred())
		})
	})
})
