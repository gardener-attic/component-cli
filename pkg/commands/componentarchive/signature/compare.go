package signature

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gardener/component-cli/ociclient"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CompareOptions struct {
	// BaseUrlFirst is the oci registry where the component is stored.
	BaseUrlFirst string
	// BaseUrlSecond is the oci registry where the component is stored.
	BaseUrlSecond string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewGetCommand shows definitions and their configuration.
func NewCompareCommand(ctx context.Context) *cobra.Command {
	opts := &CompareOptions{}
	cmd := &cobra.Command{
		Use:   "compare BASE_URL_FIRST BASE_URL_SECOND COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(4),
		Short: "compares the digests of two component descriptors",
		Long: `
compares the digests of two component descriptors.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *CompareOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}
	repoCtxFirst := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL: o.BaseUrlFirst,
	}
	repoCtxSecond := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL: o.BaseUrlSecond,
	}

	eq, uneq, err := CompareCds(repoCtxFirst, repoCtxSecond, o.ComponentName, o.Version, ociClient, "")
	if err != nil {
		return err
	}

	fmt.Println("EQUAL")
	for _, v := range eq {
		fmt.Println(v)
	}
	fmt.Println("UNEQUAL")
	for _, v := range uneq {
		fmt.Println(v)
	}

	return nil
}

func CompareCds(repoContextFirst cdv2.OCIRegistryRepository, repoContextSecond cdv2.OCIRegistryRepository, name string, version string, ociClient ociclient.Client, path string) ([]string, []string, error) {
	path = fmt.Sprintf("%s|%s:%s", path, name, version)
	equalResources := []string{}
	unEqualResources := []string{}

	cdresolver := cdoci.NewResolver(ociClient)
	firstCd, err := cdresolver.Resolve(context.TODO(), &repoContextFirst, name, version)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to to fetch component descriptor %s %s %s: %w", repoContextFirst.BaseURL, name, version, err)
	}
	secondCd, err := cdresolver.Resolve(context.TODO(), &repoContextSecond, name, version)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to to fetch component descriptor %s %s %s: %w", repoContextSecond.BaseURL, name, version, err)
	}

	for _, reference := range firstCd.ComponentReferences {
		eq, uneq, err := CompareCds(repoContextFirst, repoContextSecond, reference.ComponentName, reference.Version, ociClient, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed comparing cds %s %s: %w", reference.ComponentName, reference.Version, err)
		}
		equalResources = append(equalResources, eq...)
		unEqualResources = append(unEqualResources, uneq...)
	}

	for resIndex, res := range firstCd.Resources {
		secondRes := secondCd.Resources[resIndex]

		if reflect.DeepEqual(res.Digest, secondRes.Digest) {
			equalResources = append(equalResources, fmt.Sprintf("%s|res:%s_%s", path, res.Name, res.Version))
		} else {
			unEqualResources = append(unEqualResources, fmt.Sprintf("%s|res:%s_%s", path, res.Name, res.Version))
		}
	}
	return equalResources, unEqualResources, nil

}

func (o *CompareOptions) Complete(args []string) error {
	// todo: validate args
	o.BaseUrlFirst = args[0]
	o.BaseUrlSecond = args[1]
	o.ComponentName = args[2]
	o.Version = args[3]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}

	// TODO: disable caching
	o.OciOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OciOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OciOptions.CacheDir, err)
	}

	if len(o.BaseUrlFirst) == 0 || len(o.BaseUrlSecond) == 0 {
		return errors.New("two base url must be defined")
	}
	if len(o.ComponentName) == 0 {
		return errors.New("a component name must be defined")
	}
	if len(o.Version) == 0 {
		return errors.New("a component's Version must be defined")
	}

	return nil
}

func (o *CompareOptions) AddFlags(fs *pflag.FlagSet) {
	o.OciOptions.AddFlags(fs)
}
