package models

import (
	"sort"
	"strings"

	"github.com/JojiiOfficial/gaw"
)

//Order order/sort stuff
type Order int16

//Sorter a sorter
type Sorter struct {
	Files   []FileResponseItem
	Reverse bool
}

//NewSorter create a new sorter
func NewSorter(files []FileResponseItem) *Sorter {
	return &Sorter{
		Files:   files,
		Reverse: false,
	}
}

//Reversed Sort reversed
func (sorter *Sorter) Reversed(reversed bool) *Sorter {
	sorter.Reverse = reversed
	return sorter
}

//AvailableOrders options fo ordering
var AvailableOrders = []string{"id", "name", "size", "pubname", "created"}

//ReversedSuffixes suffixes for reversing sort
var ReversedSuffixes = []string{"r", "d"}

//Order
const (
	NoOrder Order = iota
	IDOrder
	NameOrder
	SizeOrder
	PubNameOrder
	CreatedOrder
)

//OrderFromString return order from string
func OrderFromString(str string) *Order {
	//remove direction
	str = strings.Split(str, "/")[0]

	order := NoOrder

	switch str {
	case AvailableOrders[0]:
		order = IDOrder
	case AvailableOrders[1]:
		order = NameOrder
	case AvailableOrders[2]:
		order = SizeOrder
	case AvailableOrders[3]:
		order = PubNameOrder
	case AvailableOrders[4]:
		order = CreatedOrder
	default:
		return nil
	}

	return &order
}

//IsOrderReversed return true if order should be reversed
func IsOrderReversed(str string) bool {
	if !strings.Contains(str, "/") {
		return false
	}

	direction := strings.Split(str, "/")[1]
	return gaw.IsInStringArray(direction, ReversedSuffixes)
}

//SortBy order files
func (sorter Sorter) SortBy(by Order) {
	if by == NoOrder {
		return
	}

	switch by {
	case IDOrder:
		sort.Slice(sorter.Files, sorter.sortLessID)
	case NameOrder:
		sort.Slice(sorter.Files, sorter.sortLessName)
	case SizeOrder:
		sort.Slice(sorter.Files, sorter.sortLessSize)
	case PubNameOrder:
		sort.Slice(sorter.Files, sorter.sortLessPubName)
	case CreatedOrder:
		sort.Slice(sorter.Files, sorter.sortLessCreated)
	}
}

func (sorter Sorter) sortLessID(i, j int) bool {
	if sorter.Reverse {
		i, j = j, i
	}
	return sorter.Files[i].ID < sorter.Files[j].ID
}

func (sorter Sorter) sortLessName(i, j int) bool {
	if sorter.Reverse {
		i, j = j, i
	}
	return sorter.Files[i].Name < sorter.Files[j].Name
}

func (sorter Sorter) sortLessSize(i, j int) bool {
	if sorter.Reverse {
		i, j = j, i
	}
	return sorter.Files[i].Size < sorter.Files[j].Size
}

func (sorter Sorter) sortLessPubName(i, j int) bool {
	if sorter.Reverse {
		i, j = j, i
	}
	return sorter.Files[i].PublicName < sorter.Files[j].PublicName
}

func (sorter Sorter) sortLessCreated(i, j int) bool {
	if sorter.Reverse {
		i, j = j, i
	}
	return sorter.Files[i].CreationDate.Unix() < sorter.Files[j].CreationDate.Unix()
}
