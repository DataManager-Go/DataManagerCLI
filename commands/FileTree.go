package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/fatih/color"
)

// treeItemList list of file items
type treeItemList []*libdm.FileResponseItem

// treeList a list of treeItems sorted by something (group/namespace)
type treeList map[string]treeItemList

func (cData *CommandData) renderTree(files []libdm.FileResponseItem, sOrder string) {
	// Create Treelist
	fileMap := createNamespaceTreeList(files)
	namespaces := fileMap.getNamespaces()

	// Loop namespaces and render each
	for _, namespace := range namespaces {
		renderNamespace(namespace)
		fileMap[namespace].renderNamespaceBranch(cData, sOrder, getFigureAmount(maxFileID(files))+1)
	}
}

func (treeItemList treeItemList) renderNamespaceBranch(cData *CommandData, sOrder string, indentSize int) {
	groupedList := createGroupTreeList(treeItemList)
	for k, v := range groupedList {
		renderGroupLine(k, len(v))
		v.renderFileItems(cData, 30, indentSize, sOrder)
	}
}

// renderFileItems prints files inside TreeItemList
func (treeItemList treeItemList) renderFileItems(cData *CommandData, maxFiles, indentSize int, sOrder string) {
	totalfiles := len(treeItemList)
	limit := totalfiles

	if totalfiles > maxFiles && !cData.All {
		limit = maxFiles
	}

	if !sortFiles(sOrder, treeItemList) {
		return
	}

	// Print files
	for i := 0; i < limit; i++ {
		cData.renderTreeFile(treeItemList[i], (i == totalfiles-1 && limit == totalfiles), indentSize)
	}

	// In case there are more files than allowed
	// to print, print one more line saying there
	// are some files left
	if totalfiles > limit {
		renderTreeFileBranch(fmt.Sprintf("... %d more", totalfiles-limit), true)
	}
}

// Generate treelist grouped by File groups
func createGroupTreeList(items treeItemList) treeList {
	sortByNs := make(treeList)

	for _, file := range items {
		// Set group to 'no_group' if file has no group
		if len(file.Attributes.Groups) == 0 {
			file.Attributes.Groups = []string{"no_group"}
		}

		// Loop file groups and assign the file
		// to the correct group keys
		for j := range file.Attributes.Groups {
			group := file.Attributes.Groups[j]

			it, ok := sortByNs[group]

			if !ok {
				it = make(treeItemList, 0)
				sortByNs[group] = it
			}

			sortByNs[group] = append(it, file)
		}
	}

	return sortByNs
}

// Generate treelist grouped by namespace
func createNamespaceTreeList(files []libdm.FileResponseItem) treeList {
	sortByNs := make(treeList)

	for i := range files {
		file := &files[i]
		it, ok := sortByNs[file.Attributes.Namespace]

		if !ok {
			it = make(treeItemList, 0)
			sortByNs[file.Attributes.Namespace] = it
		}

		sortByNs[file.Attributes.Namespace] = append(it, file)
	}

	return sortByNs
}

// Get all namespaces in a treelist sorted
func (treeList *treeList) getNamespaces() []string {
	var namespaces []string

	for k := range *treeList {
		namespaces = append(namespaces, k)
	}

	sort.Strings(namespaces)
	return namespaces
}

//
// --- Render functions ---- //
//

// Render file
func (cData *CommandData) renderTreeFile(file *libdm.FileResponseItem, last bool, indentSize int) {
	// Calc amount of characters of fileid
	reqIndent := getFigureAmount(file.ID)
	// Render Branch
	renderTreeFileBranch(fmt.Sprintf("[%d]%s%s", file.ID, strings.Repeat(" ", indentSize-reqIndent), formatFilename(file, 0, cData)), last)
}

// Render File-branch
func renderTreeFileBranch(name string, last bool) {
	a := "├"
	if last {
		a = "└"
	}

	fmt.Printf("          %s── %s\n", a, name)
}

// Render namespace
func renderNamespace(namespace string) {
	fmt.Println(" ─── " + color.New(color.Bold, color.FgHiYellow).Sprint(namespace))
}

// Render group
func renderGroupLine(groupName string, groupSize int) {
	c := color.New(color.FgHiBlack)

	groupName = c.Sprint(groupName)
	groupSizeStr := c.Sprint(strconv.Itoa(groupSize))

	if groupSize > 10 {
		fmt.Printf("     └── %s (%s)\n", groupName, groupSizeStr)
	} else {
		fmt.Printf("     └── %s\n", groupName)
	}
}

// Get biggest fileID of a file slice
func maxFileID(files []libdm.FileResponseItem) uint {
	if len(files) == 0 {
		return 0
	}

	// Can use first item sinc
	// we checked the slice len
	max := files[0].ID

	// Get max file
	for i := range files {
		if files[i].ID > max {
			max = files[i].ID
		}
	}

	return max
}
