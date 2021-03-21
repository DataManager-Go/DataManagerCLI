package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// Stats view users stats
func (cData *CommandData) Stats() {
	stats, err := cData.LibDM.Stats(cData.getRealNamespace())
	if err != nil {
		printResponseError(err, "retrieving stats")
		return
	}

	itemColor := color.New(color.FgHiGreen).SprintFunc()
	itemHeadingColor := color.New(color.FgHiGreen, color.Underline, color.Bold).SprintFunc()

	table := clitable.New()
	table.ColSeparator = " "
	table.Padding = 2

	table.AddRow(itemHeadingColor("Files"))
	table.AddRow(itemColor("Amount:"), stats.FilesUploaded)
	table.AddRow(itemColor("Overall size:"), units.BinarySuffix(float64(stats.TotalFileSize)))
	table.AddRow()

	table.AddRow(itemHeadingColor("Namespaces"))
	table.AddRow(itemColor("Amount:"), stats.NamespaceCount)
	table.AddRow(itemColor("Groups:"), stats.GroupCount)
	table.AddRow(itemColor("Tags:"), stats.TagCount)

	fmt.Println(table.String())
}
