// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/pkg/utils"
)

const defaultOutputPath = "./componentarchive"

const (
	OutputFormatFilesystem = "fs"
	OutputFormatTar        = "tar"
	OutputFormatTarGzip    = "tgz"
)

// ExportOptions defines all options for the export command.
type ExportOptions struct {
	// ComponentArchivePath defines the path to the component archive
	ComponentArchivePath string
	// OutputPath defines the path where the exported component archive should be written to.
	OutputPath string
	//
	OutputFormat string
}

// NewExportCommand creates a new export command that packages a component archive and
// exports is as tar or compressed tar.
func NewExportCommand(ctx context.Context) *cobra.Command {
	opts := &ExportOptions{}
	cmd := &cobra.Command{
		Use:   "export [component-archive-path] [-o output-dir/file] [-f {fs|tar|tgz}]",
		Args:  cobra.ExactArgs(1),
		Short: "Exports a component archive as defined by CTF",
		Long: `
Export command exports a component archive as defined by CTF (CNUDIE Transport Format).
If the given component-archive path points to a directory, the archive is expected to be a extracted component-archive on the filesystem.
Then it is exported as tar or optionally as compressed tar.

If the given path points to a file, the archive is read as tar or compressed tar (tar.gz) and exported as filesystem to the given location.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			if err := opts.Run(ctx, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			fmt.Printf("Successfully exported component archive to %s\n", opts.OutputPath)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the export for a component archive.
func (o *ExportOptions) Run(ctx context.Context, fs vfs.FileSystem) error {
	fileinfo, err := fs.Stat(o.ComponentArchivePath)
	if err != nil {
		return fmt.Errorf("unable to read %q: %s", o.ComponentArchivePath, err.Error())
	}

	if fileinfo.IsDir() {
		ca, err := o.caAsDir(fs)
		if err != nil {
			return err
		}
		return o.export(fs, ca, OutputFormatTar)
	} else {
		ca, err := o.caAsFile(fs)
		if err != nil {
			return nil
		}
		if err := ca.WriteToFilesystem(fs, o.OutputPath); err != nil {
			return fmt.Errorf("unable to write componant archive to %q: %s", o.OutputPath, err.Error())
		}
		return o.export(fs, ca, OutputFormatFilesystem)
	}
}

// caAsDir imports the given component archive as filesystem and outputs it as tar.
func (o *ExportOptions) caAsDir(fs vfs.FileSystem) (*ctf.ComponentArchive, error) {
	archiveFs, err := projectionfs.New(fs, o.ComponentArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create filesystem from %s: %s", o.ComponentArchivePath, err.Error())
	}
	ca, err := ctf.NewComponentArchiveFromFilesystem(archiveFs)
	if err != nil {
		return nil, fmt.Errorf("unable to parse component archive from %s: %s", o.ComponentArchivePath, err.Error())
	}
	return ca, nil
}

// caAsFile imports the given component archive as tar and outputs it as filesystem.
func (o *ExportOptions) caAsFile(fs vfs.FileSystem) (*ctf.ComponentArchive, error) {
	mimetype, err := utils.GetFileType(fs, o.ComponentArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to get mimetype of %q: %s", o.ComponentArchivePath, err.Error())
	}
	file, err := fs.Open(o.ComponentArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read component archive rom %q: %s", o.ComponentArchivePath, err.Error())
	}

	switch mimetype {
	case "application/x-gzip":
		zr, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("unable to open gzip reader: %w", err)
		}
		ca, err := ctf.NewComponentArchiveFromTarReader(zr)
		if err != nil {
			return nil, fmt.Errorf("unable to unzip componentarchive: %s", err.Error())
		}
		if err := zr.Close(); err != nil {
			return nil, fmt.Errorf("unable to close gzip reader: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("unable to close file reader: %w", err)
		}
		return ca, nil
	case "application/octet-stream": // expect that is has to be a tar
		ca, err := ctf.NewComponentArchiveFromTarReader(file)
		if err != nil {
			return nil, fmt.Errorf("unable to unzip componentarchive: %s", err.Error())
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("unable to close file reader: %w", err)
		}
		return ca, nil
	default:
		return nil, fmt.Errorf("unsupported file type %q. Expected a tar or a tar.gz", mimetype)
	}
}

func (o *ExportOptions) export(fs vfs.FileSystem, ca *ctf.ComponentArchive, defaultFormat string) error {
	if len(o.OutputFormat) == 0 {
		o.OutputFormat = defaultFormat
	}

	if o.OutputFormat != OutputFormatFilesystem && o.OutputFormat != OutputFormatTar && o.OutputFormat != OutputFormatTarGzip {
		return fmt.Errorf("unsupported output format %q, use %q, %q or %q",
			o.OutputFormat, OutputFormatFilesystem, OutputFormatTar, OutputFormatTarGzip)
	}

	if o.OutputFormat == OutputFormatFilesystem {
		if err := ca.WriteToFilesystem(fs, o.OutputPath); err != nil {
			return fmt.Errorf("unable to write componant archive to %q: %s", o.OutputPath, err.Error())
		}
		return nil
	}

	// output format is either tar or tgz

	out, err := fs.OpenFile(o.OutputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open exported file %s: %s", o.OutputPath, err.Error())
	}
	if o.OutputFormat == OutputFormatTarGzip {
		if err := ca.WriteTarGzip(out); err != nil {
			return fmt.Errorf("unable to export file to %s: %s", o.OutputPath, err.Error())
		}
	} else {
		if err := ca.WriteTar(out); err != nil {
			return fmt.Errorf("unable to export file to %s: %s", o.OutputPath, err.Error())
		}
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("unable to close file: %w", err)
	}
	return nil
}

// Complete parses the given command arguments and applies default options.
func (o *ExportOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument that contains the path to the component archive")
	}
	o.ComponentArchivePath = args[0]

	if len(o.OutputPath) == 0 {
		o.OutputPath = defaultOutputPath
	}

	return o.validate()
}

func (o *ExportOptions) validate() error {
	switch o.OutputFormat {
	case "", OutputFormatFilesystem, OutputFormatTar, OutputFormatTarGzip:
	default:
		return fmt.Errorf("unsupported output format %q, use %q, %q, %q or leave it empty to be defaulted",
			o.OutputFormat, OutputFormatFilesystem, OutputFormatTar, OutputFormatTarGzip)
	}
	return nil
}

func (o *ExportOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.OutputPath, "out", "o", "", "writes the resulting archive to the given path")
	fs.StringVarP(&o.OutputFormat, "format", "f", "", "output format of the component archive. Can be 'fs', 'tar' or 'tgz'")
}
