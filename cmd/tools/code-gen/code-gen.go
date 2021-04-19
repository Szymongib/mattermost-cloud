package main

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

var rootCmd = &cobra.Command{
	Use:   "code-gen",
	Short: "A code generation tool for Mattermost cloud",
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().String("boilerplate-file", "hack/boilerplate.go.txt", "Path to boilerplate file.")
	rootCmd.PersistentFlags().String("package", "", "Go package name to use.")
	rootCmd.MarkFlagRequired("package")

	generateStoreLocksCmd.PersistentFlags().StringArray("struct-name", []string{}, "Names of structs for which to generate code.")
	generateStoreLocksCmd.PersistentFlags().StringArray("plural-name", []string{}, "Plural names of struct for which to generate code.")
	generateStoreLocksCmd.PersistentFlags().StringArray("table-name", []string{}, "Table names of struct for which to generate code.")
	generateStoreLocksCmd.PersistentFlags().Bool("locks", true, "Whether to generate locking methods.")
	generateStoreLocksCmd.MarkFlagRequired("struct-name")

	generateSupervisorLocksCmd.PersistentFlags().StringArray("struct-name", []string{}, "Names of structs for which to generate code.")
	generateSupervisorLocksCmd.PersistentFlags().StringArray("plural-name", []string{}, "Plural names of struct for which to generate code.")
	generateSupervisorLocksCmd.PersistentFlags().StringArray("table-name", []string{}, "Table names of struct for which to generate code.")
	generateSupervisorLocksCmd.PersistentFlags().Bool("locks", true, "Whether to generate locking methods.")
	generateSupervisorLocksCmd.MarkFlagRequired("struct-name")

	generateFromReaderMethodsCmd.PersistentFlags().StringArray("struct-name", []string{}, "Names of structs for which to generate code.")
	generateFromReaderMethodsCmd.PersistentFlags().StringArray("plural-name", []string{}, "Plural names of struct for which to generate code.")
	generateFromReaderMethodsCmd.MarkFlagRequired("struct-name")

	generateFromReaderTestTemplateCmd.PersistentFlags().StringArray("struct-name", []string{}, "Names of structs for which to generate code.")
	generateFromReaderTestTemplateCmd.PersistentFlags().StringArray("plural-name", []string{}, "Plural names of struct for which to generate code.")
	generateFromReaderTestTemplateCmd.MarkFlagRequired("struct-name")

	rootCmd.AddCommand(generateStoreLocksCmd)
	rootCmd.AddCommand(generateSupervisorLocksCmd)
	rootCmd.AddCommand(generateFromReaderMethodsCmd)
	rootCmd.AddCommand(generateFromReaderTestTemplateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("command failed")
		os.Exit(1)
	}
}

type StoreGenerationData struct {
	Boilerplate string
	PackageName string
	StructName  []string
	StructNameNotExported  []string
	StructNameNotExportedPlural  []string
	PluralName []string
	TableName []string
}

var generateStoreLocksCmd = &cobra.Command{
	Use:   "generate-store",
	Short: "Generates store methods for a struct",
	RunE: func(command *cobra.Command, args []string) error {

		data, err := genDataFromFlags(command)
		if err != nil {
			return err
		}

		locks, err := generateStoreLocks(data)
		if err != nil {
			return errors.Wrap(err, "failed to generate locks")
		}

		fmt.Println(locks)

		return nil
	},
}

var generateSupervisorLocksCmd = &cobra.Command{
	Use:   "generate-supervisor",
	Short: "Generates supervisor methods for a struct",
	RunE: func(command *cobra.Command, args []string) error {

		data, err := genDataFromFlags(command)
		if err != nil {
			return err
		}

		locks, err := generateSupervisorLocks(data)
		if err != nil {
			return errors.Wrap(err, "failed to generate locks")
		}

		fmt.Println(locks)

		return nil
	},
}

var generateFromReaderMethodsCmd = &cobra.Command{
	Use:   "generate-from-reader",
	Short: "Generates from reader methods for a struct and slice of struct",
	RunE: func(command *cobra.Command, args []string) error {

		data, err := genDataFromFlags(command)
		if err != nil {
			return err
		}

		fromReaderMethods, err := generateFromReader(data)
		if err != nil {
			return errors.Wrap(err, "failed to generate from reader methods")
		}

		fmt.Println(fromReaderMethods)

		return nil
	},
}

var generateFromReaderTestTemplateCmd = &cobra.Command{
	Use:   "generate-from-reader-test",
	Short: "Generates test template  for from reader methods",
	RunE: func(command *cobra.Command, args []string) error {

		data, err := genDataFromFlags(command)
		if err != nil {
			return err
		}

		fromReaderTests, err := generateFromReaderTestTemplate(data)
		if err != nil {
			return errors.Wrap(err, "failed to generate from reader tests")
		}

		fmt.Println(fromReaderTests)

		return nil
	},
}

func genDataFromFlags(command *cobra.Command) (StoreGenerationData, error) {
	packageName, _ := command.Flags().GetString("package")

	structNames, _ := command.Flags().GetStringArray("struct-name")
	pluralNames, _ := command.Flags().GetStringArray("plural-name")
	if len(pluralNames) > 0 && len(pluralNames) < len(structNames) {
		return StoreGenerationData{}, errors.New("only all or non plural names can be provided")
	}
	if len(pluralNames) == 0 {
		for _, s := range structNames {
			pluralNames = append(pluralNames, fmt.Sprintf("%ss", s))
		}
	}
	tableNames, _ := command.Flags().GetStringArray("table-name")
	if len(tableNames) > 0 && len(tableNames) < len(structNames) {
		return StoreGenerationData{}, errors.New("only all or non table names can be provided")
	}
	if len(tableNames) == 0 {
		tableNames = structNames
	}
	boilerplate, err := loadBoilerplate(command)
	if err != nil {
		return StoreGenerationData{}, errors.Wrap(err, "failed to load boilerplate")
	}

	data := StoreGenerationData{
		Boilerplate: boilerplate,
		PackageName: packageName,
		StructName:  structNames,
		StructNameNotExported: notExportedNames(structNames),
		StructNameNotExportedPlural: notExportedNames(pluralNames),
		PluralName: pluralNames,
		TableName: tableNames,
	}

	return data, nil
}

func notExportedNames(names []string) []string {
	newNames := make([]string, len(names))

	for i, n := range names {
		newLetter := strings.ToLower(n[:1])
		newName := fmt.Sprintf("%s%s", newLetter, n[1:])
		newNames[i] = newName
	}
	return newNames
}

func loadBoilerplate(cmd *cobra.Command) (string, error) {
	bolierplateFile, _ := cmd.Flags().GetString("boilerplate-file")

	content, err := ioutil.ReadFile(bolierplateFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to read boilerplate file")
	}
	return string(content), nil
}

func generateStoreLocks(data StoreGenerationData) (string, error) {
	return createFromTemplate(data, storeLockTemplate)
}

func generateSupervisorLocks(data StoreGenerationData) (string, error) {
	return createFromTemplate(data, supervisorLocksTemplate)
}

func generateFromReader(data StoreGenerationData) (string, error) {
	return createFromTemplate(data, newFromReaderTemplate)
}

func generateFromReaderTestTemplate(data StoreGenerationData) (string, error) {
	return createFromTemplate(data, newFromReaderTestTemplate)
}

func createFromTemplate(data interface{}, rawTemplate string) (string, error) {
	tmpl, err := template.New("").Parse(rawTemplate)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
