package table

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ddkwork/golibrary/stream"

	"cogentcore.org/cogent/ai/pkg/tree"
)

type Provider[T any] interface {
	//Model[*Node[T]]
	//SetTable(table *tree.Node[T]) //todo
	RootData() []T
	SetRootData(data []T)
	DragKey() string
	//DragSVG() *SVG
	//DropShouldMoveData(from, to *unison.Table[*Node[T]]) bool
	//ProcessDropData(from, to *unison.Table[*Node[T]])
	//AltDropSupport() *AltDropSupport
	ItemNames() (singular, plural string)
	//Headers() []unison.TableColumnHeader[*Node[T]]
	//SyncHeader(headers []unison.TableColumnHeader[*Node[T]])
	ColumnIDs() []int
	HierarchyColumnID() int
	ExcessWidthColumnID() int
	//ContextMenuItems() []ContextMenuItem
	//OpenEditor(owner Rebuildable, table *unison.Table[*Node[T]])
	//CreateItem(owner Rebuildable, table *unison.Table[*Node[T]], variant ItemVariant)

	RefKey() string
	AllTags() []string
	CellFromCellData()
}

type (
	RowConstraint[T any] interface {
		comparable
	}
	Model[T RowConstraint[T]] interface {
		RootRowCount() int
		RootRows() []T
		SetRootRows(rows []T)
	}
	SimpleModel[T RowConstraint[T]] struct{ roots []T }
)

func (m *SimpleModel[T]) RootRowCount() int    { return len(m.roots) }
func (m *SimpleModel[T]) RootRows() []T        { return m.roots }
func (m *SimpleModel[T]) SetRootRows(rows []T) { m.roots = rows }

type RowData struct {
	*tree.Node[*RowData]
}

func NewRowData() tree.Provider[*RowData] {
	return &RowData{}
}

//
//func (n *RowData[T]) ColumnCell(row, col int, foreground, background unison.Ink, _, _, _ bool) unison.Paneler {
//	var cellData gurps.CellData
//	n.dataAsNode.CellData(n.table.Columns[col].ID, &cellData)
//	width := n.table.CellWidth(row, col)
//	if n.cellCache[col].Matches(width, &cellData) {
//		applyInkRecursively(n.cellCache[col].Panel.AsPanel(), foreground, background)
//		return n.cellCache[col].Panel
//	}
//	c := n.CellFromCellData(&cellData, width, foreground, background)
//	n.cellCache[col] = &CellCache{
//		Panel: c,
//		Data:  cellData,
//		Width: width,
//	}
//	return c
//}

//func CollectUUIDsFromRow[T RowConstraint[T]](node T, ids map[uuid.UUID]bool) {
//	ids[node.UUID()] = true
//	for _, child := range node.Children() {
//
//		CollectUUIDsFromRow(AsNode(child), ids)
//	}
//}
//
//func AsNode[T any](in T) Node[T] { return any(in).(Node[T]) }//not need

func FormatDataForEdit(rowObjectStruct any) (rowData []string) { //todo need merge into formatData method
	rowData = make([]string, 0)
	valueOf := reflect.ValueOf(rowObjectStruct)
	typeOf := reflect.Indirect(valueOf)
	if typeOf.Kind() != reflect.Struct {
		rowData = append(rowData, fmt.Sprint(rowObjectStruct))
		return
	}
	fields := reflect.VisibleFields(typeOf.Type())
	for i, field := range fields {
		field = field
		//mylog.Struct(field)
		v := valueOf.Field(i).Interface()
		switch t := v.(type) {
		case string:
			rowData = append(rowData, t)
		case int64:
			rowData = append(rowData, fmt.Sprint(t))
		case int:
			rowData = append(rowData, fmt.Sprint(t))
		case time.Time:
			rowData = append(rowData, stream.FormatTime(t))
		case time.Duration:
			rowData = append(rowData, fmt.Sprint(t))
		case reflect.Kind:
			rowData = append(rowData, t.String())
		case bool: // todo 不应该支持？数据库是否会有这种情况？
			rowData = append(rowData, fmt.Sprint(t))
		default: // any
			rowData = append(rowData, fmt.Sprint(t))
		}
	}
	return
}

const containerMarker = "\000"

// ItemVariant holds the type of item variant to create.
type ItemVariant int

// Possible values for ItemVariant.
const (
	NoItemVariant ItemVariant = iota
	ContainerItemVariant
	AlternateItemVariant
)

/*
func flexibleLess(s1, s2 string) bool {
	c1 := strings.HasPrefix(s1, containerMarker)
	c2 := strings.HasPrefix(s2, containerMarker)
	if c1 != c2 {
		return c1
	}
	if c1 {
		s1 = s1[1:]
	}
	if c2 {
		s2 = s2[1:]
	}
	if n1, err := fxp.FromString(s1); err == nil {
		var n2 fxp.Int
		if n2, err = fxp.FromString(s2); err == nil {
			return n1 < n2
		}
	}
	return txt.NaturalLess(s1, s2, true)
}

// OpenEditor opens an editor for each selected row in the table.
func OpenEditor[T gurps.NodeTypes](table *unison.Table[*Node[T]], edit func(item T)) {
	var zero T
	selection := table.SelectedRows(false)
	if len(selection) > 4 {
		if unison.QuestionDialog(i18n.Text("Are you sure you want to open all of these?"),
			fmt.Sprintf(i18n.Text("%d editors will be opened."), len(selection))) != unison.ModalResponseOK {
			return
		}
	}
	for _, row := range selection {
		if data := row.Data(); data != zero {
			edit(data)
		}
	}
}

// DeleteSelection removes the selected nodes from the table.
func DeleteSelection[T gurps.NodeTypes](table *unison.Table[*Node[T]], recordUndo bool) {
	if provider, ok := any(table.Model).(TableProvider[T]); ok && !table.IsFiltered() && table.HasSelection() {
		sel := table.SelectedRows(true)
		ids := make(map[uuid.UUID]bool, len(sel))
		list := make([]T, 0, len(sel))
		var zero T
		for _, row := range sel {
			unison.CollectUUIDsFromRow(row, ids)
			if target := row.Data(); target != zero {
				list = append(list, target)
			}
		}
		if !CloseUUID(ids) {
			return
		}
		var undo *unison.UndoEdit[*TableUndoEditData[T]]
		var mgr *unison.UndoManager
		if recordUndo {
			if mgr = unison.UndoManagerFor(table); mgr != nil {
				undo = &unison.UndoEdit[*TableUndoEditData[T]]{
					ID:         unison.NextUndoID(),
					EditName:   i18n.Text("Delete Selection"),
					UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
					RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
					AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
					BeforeData: NewTableUndoEditData(table),
				}
			}
		}
		needSet := false
		topLevelData := provider.RootData()
		for _, target := range list {
			parent := gurps.AsNode(target).Parent()
			if parent == zero {
				for i, one := range topLevelData {
					if one == target {
						topLevelData = slices.Delete(topLevelData, i, i+1)
						needSet = true
						break
					}
				}
			} else {
				pNode := gurps.AsNode(parent)
				children := pNode.NodeChildren()
				for i, one := range children {
					if one == target {
						pNode.SetChildren(slices.Delete(children, i, i+1))
						break
					}
				}
			}
		}
		if needSet {
			provider.SetRootData(topLevelData)
		}
		if recordUndo && mgr != nil && undo != nil {
			undo.AfterData = NewTableUndoEditData(table)
			mgr.Add(undo)
		}
		if builder := unison.AncestorOrSelf[Rebuildable](table); builder != nil {
			builder.Rebuild(true)
		}
	}
}

// DuplicateSelection duplicates the selected nodes in the table.
func DuplicateSelection[T gurps.NodeTypes](table *unison.Table[*Node[T]]) {
	if provider, ok := any(table.Model).(TableProvider[T]); ok && !table.IsFiltered() && table.HasSelection() {
		var undo *unison.UndoEdit[*TableUndoEditData[T]]
		mgr := unison.UndoManagerFor(table)
		if mgr != nil {
			undo = &unison.UndoEdit[*TableUndoEditData[T]]{
				ID:         unison.NextUndoID(),
				EditName:   i18n.Text("Duplicate Selection"),
				UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
				RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
				AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
				BeforeData: NewTableUndoEditData(table),
			}
		}
		var zero T
		needSet := false
		topLevelData := provider.RootData()
		sel := table.SelectedRows(true)
		selMap := make(map[uuid.UUID]bool, len(sel))
		for _, row := range sel {
			if target := row.Data(); target != zero {
				tData := gurps.AsNode(target)
				parent := tData.Parent()
				clone := tData.Clone(tData.OwningEntity(), parent, false)
				selMap[gurps.AsNode(clone).UUID()] = true
				if parent == zero {
					for i, child := range topLevelData {
						if child == target {
							topLevelData = slices.Insert(topLevelData, i+1, clone)
							needSet = true
							break
						}
					}
				} else {
					pNode := gurps.AsNode(parent)
					children := pNode.NodeChildren()
					for i, child := range children {
						if child == target {
							pNode.SetChildren(slices.Insert(children, i+1, clone))
							break
						}
					}
				}
			}
		}
		if needSet {
			provider.SetRootData(topLevelData)
		}
		table.SyncToModel()
		table.SetSelectionMap(selMap)
		if mgr != nil && undo != nil {
			undo.AfterData = NewTableUndoEditData(table)
			mgr.Add(undo)
		}
		if builder := unison.AncestorOrSelf[Rebuildable](table); builder != nil {
			builder.Rebuild(true)
		}
	}
}

// CopyRowsTo copies the provided rows to the target table.
func CopyRowsTo[T gurps.NodeTypes](table *unison.Table[*Node[T]], rows []*Node[T], postProcessor func(rows []*Node[T]), recordUndo bool) {
	if table == nil || table.IsFiltered() {
		return
	}
	rows = slices.Clone(rows)
	for j, row := range rows {
		rows[j] = row.CloneForTarget(table, nil)
	}
	var undo *unison.UndoEdit[*TableUndoEditData[T]]
	var mgr *unison.UndoManager
	if recordUndo {
		if mgr = unison.UndoManagerFor(table); mgr != nil {
			undo = &unison.UndoEdit[*TableUndoEditData[T]]{
				ID:         unison.NextUndoID(),
				EditName:   fmt.Sprintf(i18n.Text("Insert %s"), gurps.AsNode(rows[0].Data()).Kind()),
				UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
				RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
				AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
				BeforeData: NewTableUndoEditData(table),
			}
		}
	}
	table.SetRootRows(append(slices.Clone(table.RootRows()), rows...))
	selMap := make(map[uuid.UUID]bool, len(rows))
	for _, row := range rows {
		selMap[row.UUID()] = true
	}
	table.SetSelectionMap(selMap)
	if postProcessor != nil {
		postProcessor(rows)
	}
	table.ScrollRowCellIntoView(table.LastSelectedRowIndex(), 0)
	table.ScrollRowCellIntoView(table.FirstSelectedRowIndex(), 0)
	if recordUndo && mgr != nil && undo != nil {
		undo.AfterData = NewTableUndoEditData(table)
		mgr.Add(undo)
	}
	unison.Ancestor[Rebuildable](table).Rebuild(true)
}

// DisableSorting disables the sorting capability in the table headers.
func DisableSorting[T unison.TableRowConstraint[T]](headers []unison.TableColumnHeader[T]) []unison.TableColumnHeader[T] {
	for _, header := range headers {
		state := header.SortState()
		state.Sortable = false
		header.SetSortState(state)
	}
	return headers
}

*/

// FindRowIndexByID returns the row index of the row with the given ID in the given table.
//func FindRowIndexByID[T any](table *unison.Table[*Node[T]], id uuid.UUID) int {
//	_, i := rowIndex(id, 0, table.RootRows())
//	return i
//}

//func rowIndex[T any](id uuid.UUID, startIndex int, rows []*Node[T]) (updatedStartIndex, result int) {
//	for _, row := range rows {
//		if id == row.dataAsNode.UUID() {
//			return 0, startIndex
//		}
//		startIndex++
//		if row.IsOpen() {
//			if startIndex, result = rowIndex(id, startIndex, row.Children()); result != -1 {
//				return 0, result
//			}
//		}
//	}
//	return startIndex, -1
//}

// InsertItems into a table.
//func InsertItems[T any](owner Rebuildable, table *unison.Table[*Node[T]], topList func() []T, setTopList func([]T), rowData func(table *unison.Table[*Node[T]]) []*Node[T], items ...T) {
//	if len(items) == 0 {
//		return
//	}
//	var undo *unison.UndoEdit[*TableUndoEditData[T]]
//	mgr := unison.UndoManagerFor(table)
//	if mgr != nil {
//		undo = &unison.UndoEdit[*TableUndoEditData[T]]{
//			ID:         unison.NextUndoID(),
//			EditName:   fmt.Sprintf(i18n.Text("Insert %s"), gurps.AsNode(items[0]).Kind()),
//			UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
//			RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
//			AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
//			BeforeData: NewTableUndoEditData(table),
//		}
//	}
//	var target, zero T
//	i := table.FirstSelectedRowIndex()
//	if i != -1 {
//		row := table.RowFromIndex(i)
//		if target = row.Data(); target != zero {
//			if row.CanHaveChildren() {
//				// Target is container, append to end of that container
//				SetParents(items, target)
//				row.dataAsNode.SetChildren(append(row.dataAsNode.NodeChildren(), items...))
//			} else {
//				// Target isn't a container. If it has a parent, insert after the target within that parent.
//				parent := row.Parent()
//				if parentData := parent.Data(); parentData != zero {
//					SetParents(items, parentData)
//					children := parent.dataAsNode.NodeChildren()
//					parent.dataAsNode.SetChildren(slices.Insert(children, slices.Index(children, target)+1, items...))
//				} else {
//					// Otherwise, insert after the target within the top-level list.
//					SetParents(items, zero)
//					list := topList()
//					setTopList(slices.Insert(list, slices.Index(list, target)+1, items...))
//				}
//			}
//		}
//	}
//	if target == zero {
//		// There was no selection, so append to the end of the top-level list.
//		SetParents(items, zero)
//		setTopList(append(topList(), items...))
//	}
//	MarkModified(table)
//	table.SetRootRows(rowData(table))
//	table.ValidateScrollRoot()
//	table.RequestFocus()
//	selMap := make(map[uuid.UUID]bool)
//	for _, item := range items {
//		selMap[gurps.AsNode(item).UUID()] = true
//	}
//	table.SetSelectionMap(selMap)
//	table.ScrollRowCellIntoView(table.LastSelectedRowIndex(), 0)
//	table.ScrollRowCellIntoView(table.FirstSelectedRowIndex(), 0)
//	if mgr != nil && undo != nil {
//		undo.AfterData = NewTableUndoEditData(table)
//		mgr.Add(undo)
//	}
//	owner.Rebuild(true)
//}

// SetParents of each item.
//func SetParents[T any](items []T, parent T) {
//	for _, item := range items {
//		AsNode(item).SetParent(parent)
//	}
//}

//// CountTableRows returns the number of table rows, including all descendants, whether open or not.
//func CountTableRows[T RowConstraint[T]](rows []T) int {
//	count := len(rows)
//	for _, row := range rows {
//		if row.CanHaveChildren() {
//			count += CountTableRows(row.Children())
//		}
//	}
//	return count
//}
//
//// RowContainsRow returns true if 'descendant' is in fact a descendant of 'ancestor'.
//func RowContainsRow[T RowConstraint[T]](ancestor, descendant T) bool {
//	var zero T
//	for descendant != zero && descendant != ancestor {
//		descendant = descendant.Parent()
//	}
//	return descendant == ancestor
//}

/*

func (t *Table[T]) ApplyFilter(filter func(row T) bool) {
	if filter == nil {
		if t.filteredRows == nil {
			return
		}
		t.filteredRows = nil
	} else {
		t.filteredRows = make([]T, 0)
		for _, row := range t.Model.RootRows() {
			t.applyFilter(row, filter)
		}
	}
	t.SyncToModel()
	if t.header != nil && t.header.HasSort() {
		t.header.ApplySort()
	}
}

func (t *Table[T]) applyFilter(row T, filter func(row T) bool) {
	if !filter(row) {
		t.filteredRows = append(t.filteredRows, row)
	}
	if row.CanHaveChildren() {
		for _, child := range row.Children() {
			t.applyFilter(child, filter)
		}
	}
}






var zeroUUID = uuid.UUID{}

// TableDragData holds the data from a table row drag.
type TableDragData[T TableRowConstraint[T]] struct {
	Table *Table[T]
	Rows  []T
}

// ColumnInfo holds column information.
type ColumnInfo struct {
	ID          int
	Current     float32
	Minimum     float32
	Maximum     float32
	AutoMinimum float32
	AutoMaximum float32
}

type tableCache[T TableRowConstraint[T]] struct {
	row    T
	parent int
	depth  int
	height float32
}

type tableHitRect struct {
	//Rect
	handler func()
}

// DefaultTableTheme holds the default TableTheme values for Tables. Modifying this data will not alter existing Tables,
// but will alter any Tables created in the future.
var DefaultTableTheme = TableTheme{
	//BackgroundInk:          ContentColor,
	//OnBackgroundInk:        OnContentColor,
	//BandingInk:             BandingColor,
	//OnBandingInk:           OnBandingColor,
	//InteriorDividerInk:     InteriorDividerColor,
	//SelectionInk:           SelectionColor,
	//OnSelectionInk:         OnSelectionColor,
	//InactiveSelectionInk:   InactiveSelectionColor,
	//OnInactiveSelectionInk: OnInactiveSelectionColor,
	//IndirectSelectionInk:   IndirectSelectionColor,
	//OnIndirectSelectionInk: OnIndirectSelectionColor,
	//Padding:                NewUniformInsets(4),
	HierarchyIndent:   16,
	MinimumRowHeight:  16,
	ColumnResizeSlop:  4,
	ShowRowDivider:    true,
	ShowColumnDivider: true,
}

// TableTheme holds theming data for a Table.
type TableTheme struct {
	//BackgroundInk          Ink
	//OnBackgroundInk        Ink
	//BandingInk             Ink
	//OnBandingInk           Ink
	//InteriorDividerInk     Ink
	//SelectionInk           Ink
	//OnSelectionInk         Ink
	//InactiveSelectionInk   Ink
	//OnInactiveSelectionInk Ink
	//IndirectSelectionInk   Ink
	//OnIndirectSelectionInk Ink
	//Padding                Insets
	HierarchyColumnID int
	HierarchyIndent   float32
	MinimumRowHeight  float32
	ColumnResizeSlop  float32
	ShowRowDivider    bool
	ShowColumnDivider bool
}

// Table provides a control that can display data in columns and rows.
type Table[T TableRowConstraint[T]] struct {
	//Panel
	TableTheme
	SelectionChangedCallback func()
	DoubleClickCallback      func()
	DragRemovedRowsCallback  func() // Called whenever a drag removes one or more rows from a model, but only if the source and destination tables were different.
	DropOccurredCallback     func() // Called whenever a drop occurs that modifies the model.
	Columns                  []ColumnInfo
	Model                    TableModel[T]
	filteredRows             []T // Note that we use the difference between nil and an empty slice here
	//header                   *TableHeader[T]
	selMap    map[uuid.UUID]bool
	selAnchor uuid.UUID
	lastSel   uuid.UUID
	hitRects  []tableHitRect
	rowCache  []tableCache[T]
	//lastMouseEnterCellPanel  *Panel
	//lastMouseDownCellPanel   *Panel
	interactionRow           int
	interactionColumn        int
	lastMouseMotionRow       int
	lastMouseMotionColumn    int
	startRow                 int
	endBeforeRow             int
	columnResizeStart        float32
	columnResizeBase         float32
	columnResizeOverhead     float32
	PreventUserColumnResize  bool
	awaitingSizeColumnsToFit bool
	awaitingSyncToModel      bool
	selNeedsPrune            bool
	wasDragged               bool
	dividerDrag              bool
}

// NewTable creates a new Table control.
func NewTable[T TableRowConstraint[T]](model TableModel[T]) *Table[T] {
	t := &Table[T]{
		TableTheme:            DefaultTableTheme,
		Model:                 model,
		selMap:                make(map[uuid.UUID]bool),
		interactionRow:        -1,
		interactionColumn:     -1,
		lastMouseMotionRow:    -1,
		lastMouseMotionColumn: -1,
	}
	//t.Self = t
	//t.SetFocusable(true)
	//t.SetSizer(t.DefaultSizes)
	//t.GainedFocusCallback = t.DefaultFocusGained
	//t.DrawCallback = t.DefaultDraw
	//t.UpdateCursorCallback = t.DefaultUpdateCursorCallback
	//t.UpdateTooltipCallback = t.DefaultUpdateTooltipCallback
	//t.MouseMoveCallback = t.DefaultMouseMove
	//t.MouseDownCallback = t.DefaultMouseDown
	//t.MouseDragCallback = t.DefaultMouseDrag
	//t.MouseUpCallback = t.DefaultMouseUp
	//t.MouseEnterCallback = t.DefaultMouseEnter
	//t.MouseExitCallback = t.DefaultMouseExit
	//t.KeyDownCallback = t.DefaultKeyDown
	//t.InstallCmdHandlers(SelectAllItemID, AlwaysEnabled, func(_ any) { t.SelectAll() })
	t.wasDragged = false
	return t
}

// ColumnIndexForID returns the column index with the given ID, or -1 if not found.
func (t *Table[T]) ColumnIndexForID(id int) int {
	for i, c := range t.Columns {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// SetDrawRowRange sets a restricted range for sizing and drawing the table. This is intended primarily to be able to
// draw different sections of the table on separate pages of a display and should not be used for anything requiring
// interactivity.
func (t *Table[T]) SetDrawRowRange(start, endBefore int) {
	t.startRow = start
	t.endBeforeRow = endBefore
}

// ClearDrawRowRange clears any restricted range for sizing and drawing the table.
func (t *Table[T]) ClearDrawRowRange() {
	t.startRow = 0
	t.endBeforeRow = 0
}

// CurrentDrawRowRange returns the range of rows that are considered for sizing and drawing.
func (t *Table[T]) CurrentDrawRowRange() (start, endBefore int) {
	if t.startRow < t.endBeforeRow && t.startRow >= 0 && t.endBeforeRow <= len(t.rowCache) {
		return t.startRow, t.endBeforeRow
	}
	return 0, len(t.rowCache)
}

// DefaultDraw provides the default drawing.
//func (t *Table[T]) DefaultDraw(canvas *Canvas, dirty Rect) {
//	selectionInk := t.SelectionInk
//	if !t.Focused() {
//		selectionInk = t.InactiveSelectionInk
//	}
//
//	canvas.DrawRect(dirty, t.BackgroundInk.Paint(canvas, dirty, paintstyle.Fill))
//
//	var insets Insets
//	if border := t.Border(); border != nil {
//		insets = border.Insets()
//	}
//
//	var firstCol int
//	x := insets.Left
//	for i := range t.Columns {
//		x1 := x + t.Columns[i].Current
//		if t.ShowColumnDivider {
//			x1++
//		}
//		if x1 >= dirty.X {
//			break
//		}
//		x = x1
//		firstCol = i + 1
//	}
//
//	startRow, endBeforeRow := t.CurrentDrawRowRange()
//	y := insets.Top
//	for i := startRow; i < endBeforeRow; i++ {
//		y1 := y + t.rowCache[i].height
//		if t.ShowRowDivider {
//			y1++
//		}
//		if y1 >= dirty.Y {
//			break
//		}
//		y = y1
//		startRow = i + 1
//	}
//
//	lastY := dirty.Bottom()
//	rect := dirty
//	rect.Y = y
//	for r := startRow; r < endBeforeRow && rect.Y < lastY; r++ {
//		rect.Height = t.rowCache[r].height
//		if t.IsRowOrAnyParentSelected(r) {
//			if t.IsRowSelected(r) {
//				canvas.DrawRect(rect, selectionInk.Paint(canvas, rect, paintstyle.Fill))
//			} else {
//				canvas.DrawRect(rect, t.IndirectSelectionInk.Paint(canvas, rect, paintstyle.Fill))
//			}
//		} else if r%2 == 1 {
//			canvas.DrawRect(rect, t.BandingInk.Paint(canvas, rect, paintstyle.Fill))
//		}
//		rect.Y += t.rowCache[r].height
//		if t.ShowRowDivider && r != endBeforeRow-1 {
//			rect.Height = 1
//			canvas.DrawRect(rect, t.InteriorDividerInk.Paint(canvas, rect, paintstyle.Fill))
//			rect.Y++
//		}
//	}
//
//	if t.ShowColumnDivider {
//		rect = dirty
//		rect.X = x
//		rect.Width = 1
//		for c := firstCol; c < len(t.Columns)-1; c++ {
//			rect.X += t.Columns[c].Current
//			canvas.DrawRect(rect, t.InteriorDividerInk.Paint(canvas, rect, paintstyle.Fill))
//			rect.X++
//		}
//	}
//
//	rect = dirty
//	rect.Y = y
//	lastX := dirty.Right()
//	t.hitRects = nil
//	for r := startRow; r < endBeforeRow && rect.Y < lastY; r++ {
//		rect.X = x
//		rect.Height = t.rowCache[r].height
//		for c := firstCol; c < len(t.Columns) && rect.X < lastX; c++ {
//			fg, bg, selected, indirectlySelected, focused := t.cellParams(r, c)
//			rect.Width = t.Columns[c].Current
//			cellRect := rect.Inset(t.Padding)
//			row := t.rowCache[r].row
//			if t.Columns[c].ID == t.HierarchyColumnID {
//				if row.CanHaveChildren() {
//					const disclosureIndent = 2
//					disclosureSize := min(t.HierarchyIndent, t.MinimumRowHeight) - disclosureIndent*2
//					canvas.Save()
//					left := cellRect.X + t.HierarchyIndent*float32(t.rowCache[r].depth) + disclosureIndent
//					top := cellRect.Y + (t.MinimumRowHeight-disclosureSize)/2
//					dSize := Size{Width: disclosureSize, Height: disclosureSize}
//					t.hitRects = append(t.hitRects,
//						t.newTableHitRect(Rect{Point: Point{X: left, Y: top}, Size: dSize}, row))
//					canvas.Translate(left, top)
//					if row.IsOpen() {
//						offset := disclosureSize / 2
//						canvas.Translate(offset, offset)
//						canvas.Rotate(90)
//						canvas.Translate(-offset, -offset)
//					}
//					canvas.DrawPath(CircledChevronRightSVG.PathForSize(dSize),
//						fg.Paint(canvas, cellRect, paintstyle.Fill))
//					canvas.Restore()
//				}
//				indent := t.HierarchyIndent*float32(t.rowCache[r].depth+1) + t.Padding.Left
//				cellRect.X += indent
//				cellRect.Width -= indent
//			}
//			cell := row.ColumnCell(r, c, fg, bg, selected, indirectlySelected, focused).AsPanel()
//			t.installCell(cell, cellRect)
//			canvas.Save()
//			canvas.Translate(cellRect.X, cellRect.Y)
//			cellRect.X = 0
//			cellRect.Y = 0
//			cell.Draw(canvas, cellRect)
//			t.uninstallCell(cell)
//			canvas.Restore()
//			rect.X += t.Columns[c].Current
//			if t.ShowColumnDivider {
//				rect.X++
//			}
//		}
//		rect.Y += t.rowCache[r].height
//		if t.ShowRowDivider {
//			rect.Y++
//		}
//	}
//}

//func (t *Table[T]) cellParams(row, _ int) (fg, bg Ink, selected, indirectlySelected, focused bool) {
//	focused = t.Focused()
//	selected = t.IsRowSelected(row)
//	indirectlySelected = !selected && t.IsRowOrAnyParentSelected(row)
//	switch {
//	case selected && focused:
//		fg = t.OnSelectionInk
//		bg = t.SelectionInk
//	case selected:
//		fg = t.OnInactiveSelectionInk
//		bg = t.InactiveSelectionInk
//	case indirectlySelected:
//		fg = t.OnIndirectSelectionInk
//		bg = t.IndirectSelectionInk
//	case row%2 == 1:
//		fg = t.OnBandingInk
//		bg = t.BandingInk
//	default:
//		fg = t.OnBackgroundInk
//		bg = t.BackgroundInk
//	}
//	return fg, bg, selected, indirectlySelected, focused
//}

//func (t *Table[T]) cell(row, col int) *Panel {
//	fg, bg, selected, indirectlySelected, focused := t.cellParams(row, col)
//	return t.rowCache[row].row.ColumnCell(row, col, fg, bg, selected, indirectlySelected, focused).AsPanel()
//}
//
//func (t *Table[T]) installCell(cell *Panel, frame Rect) {
//	cell.SetFrameRect(frame)
//	cell.ValidateLayout()
//	cell.parent = t.AsPanel()
//}
//
//func (t *Table[T]) uninstallCell(cell *Panel) {
//	cell.parent = nil
//}

// RowHeights returns the heights of each row.
func (t *Table[T]) RowHeights() []float32 {
	heights := make([]float32, len(t.rowCache))
	for i := range t.rowCache {
		heights[i] = t.rowCache[i].height
	}
	return heights
}

// OverRow returns the row index that the y coordinate is over, or -1 if it isn't over any row.
func (t *Table[T]) OverRow(y float32) int {
	//var insets Insets
	//if border := t.Border(); border != nil {
	//	insets = border.Insets()
	//}
	//end := insets.Top
	//for i := range t.rowCache {
	//	start := end
	//	end += t.rowCache[i].height
	//	if t.ShowRowDivider {
	//		end++
	//	}
	//	if y >= start && y < end {
	//		return i
	//	}
	//}
	return -1
}

// OverColumn returns the column index that the x coordinate is over, or -1 if it isn't over any column.
func (t *Table[T]) OverColumn(x float32) int {
	//var insets Insets
	//if border := t.Border(); border != nil {
	//	insets = border.Insets()
	//}
	//end := insets.Left
	//for i := range t.Columns {
	//	start := end
	//	end += t.Columns[i].Current
	//	if t.ShowColumnDivider {
	//		end++
	//	}
	//	if x >= start && x < end {
	//		return i
	//	}
	//}
	return -1
}

// OverColumnDivider returns the column index of the column divider that the x coordinate is over, or -1 if it isn't
// over any column divider.
func (t *Table[T]) OverColumnDivider(x float32) int {
	if len(t.Columns) < 2 {
		return -1
	}
	//var insets Insets
	//if border := t.Border(); border != nil {
	//	insets = border.Insets()
	//}
	//pos := insets.Left
	//for i := range t.Columns[:len(t.Columns)-1] {
	//	pos += t.Columns[i].Current
	//	if t.ShowColumnDivider {
	//		pos++
	//	}
	//	if xmath.Abs(pos-x) < t.ColumnResizeSlop {
	//		return i
	//	}
	//}
	return -1
}

// CellWidth returns the current width of a given cell.
func (t *Table[T]) CellWidth(row, col int) float32 {
	if row < 0 || col < 0 || row >= len(t.rowCache) || col >= len(t.Columns) {
		return 0
	}
	//width := t.Columns[col].Current - (t.Padding.Left + t.Padding.Right)
	//if t.Columns[col].ID == t.HierarchyColumnID {
	//	width -= t.HierarchyIndent*float32(t.rowCache[row].depth+1) + t.Padding.Left
	//}
	//return width
	return 0
}

// ColumnEdges returns the x-coordinates of the left and right sides of the column.
func (t *Table[T]) ColumnEdges(col int) (left, right float32) {
	if col < 0 || col >= len(t.Columns) {
		return 0, 0
	}
	//var insets Insets
	//if border := t.Border(); border != nil {
	//	insets = border.Insets()
	//}
	//left = insets.Left
	//for c := 0; c < col; c++ {
	//	left += t.Columns[c].Current
	//	if t.ShowColumnDivider {
	//		left++
	//	}
	//}
	//right = left + t.Columns[col].Current
	//left += t.Padding.Left
	//right -= t.Padding.Right
	//if t.Columns[col].ID == t.HierarchyColumnID {
	//	left += t.HierarchyIndent + t.Padding.Left
	//}
	//if right < left {
	//	right = left
	//}
	return left, right
}

// CellFrame returns the frame of the given cell.
//func (t *Table[T]) CellFrame(row, col int) Rect {
//	if row < 0 || col < 0 || row >= len(t.rowCache) || col >= len(t.Columns) {
//		return Rect{}
//	}
//	var insets Insets
//	if border := t.Border(); border != nil {
//		insets = border.Insets()
//	}
//	x := insets.Left
//	for c := 0; c < col; c++ {
//		x += t.Columns[c].Current
//		if t.ShowColumnDivider {
//			x++
//		}
//	}
//	y := insets.Top
//	for r := 0; r < row; r++ {
//		y += t.rowCache[r].height
//		if t.ShowRowDivider {
//			y++
//		}
//	}
//	rect := Rect{
//		Point: Point{X: x, Y: y},
//		Size:  Size{Width: t.Columns[col].Current, Height: t.rowCache[row].height},
//	}.Inset(t.Padding)
//	if t.Columns[col].ID == t.HierarchyColumnID {
//		indent := t.HierarchyIndent*float32(t.rowCache[row].depth+1) + t.Padding.Left
//		rect.X += indent
//		rect.Width -= indent
//		if rect.Width < 1 {
//			rect.Width = 1
//		}
//	}
//	return rect
//}

// RowFrame returns the frame of the row.
//func (t *Table[T]) RowFrame(row int) Rect {
//	if row < 0 || row >= len(t.rowCache) {
//		return Rect{}
//	}
//	rect := t.ContentRect(false)
//	for i := 0; i < row; i++ {
//		rect.Y += t.rowCache[i].height
//		if t.ShowRowDivider {
//			rect.Y++
//		}
//	}
//	rect.Height = t.rowCache[row].height
//	return rect
//}

//func (t *Table[T]) newTableHitRect(rect Rect, row T) tableHitRect {
//	return tableHitRect{
//		Rect: rect,
//		handler: func() {
//			open := !row.IsOpen()
//			row.SetOpen(open)
//			t.SyncToModel()
//			if !open {
//				t.PruneSelectionOfUndisclosedNodes()
//			}
//		},
//	}
//}

// DefaultFocusGained provides the default focus gained handling.
func (t *Table[T]) DefaultFocusGained() {
	switch {
	//case t.interactionRow != -1:
	//	t.ScrollRowIntoView(t.interactionRow)
	//case t.lastMouseMotionRow != -1:
	//	t.ScrollRowIntoView(t.lastMouseMotionRow)
	//default:
	//	t.ScrollIntoView()
	}
	//t.MarkForRedraw()
}

// DefaultUpdateCursorCallback provides the default cursor update handling.
//func (t *Table[T]) DefaultUpdateCursorCallback(where Point) *Cursor {
//	if !t.PreventUserColumnResize {
//		if over := t.OverColumnDivider(where.X); over != -1 {
//			if t.Columns[over].Minimum <= 0 || t.Columns[over].Minimum < t.Columns[over].Maximum {
//				return ResizeHorizontalCursor()
//			}
//		}
//	}
//	if row := t.OverRow(where.Y); row != -1 {
//		if col := t.OverColumn(where.X); col != -1 {
//			cell := t.cell(row, col)
//			if cell.HasInSelfOrDescendants(func(p *Panel) bool { return p.UpdateCursorCallback != nil }) {
//				var cursor *Cursor
//				rect := t.CellFrame(row, col)
//				t.installCell(cell, rect)
//				where = where.Sub(rect.Point)
//				target := cell.PanelAt(where)
//				for target != t.AsPanel() {
//					if target.UpdateCursorCallback == nil {
//						target = target.parent
//					} else {
//						toolbox.Call(func() { cursor = target.UpdateCursorCallback(cell.PointTo(where, target)) })
//						break
//					}
//				}
//				t.uninstallCell(cell)
//				return cursor
//			}
//		}
//	}
//	return nil
//}
//
//// DefaultUpdateTooltipCallback provides the default tooltip update handling.
//func (t *Table[T]) DefaultUpdateTooltipCallback(where Point, avoid Rect) Rect {
//	if row := t.OverRow(where.Y); row != -1 {
//		if col := t.OverColumn(where.X); col != -1 {
//			cell := t.cell(row, col)
//			if cell.HasInSelfOrDescendants(func(p *Panel) bool { return p.UpdateTooltipCallback != nil || p.Tooltip != nil }) {
//				rect := t.CellFrame(row, col)
//				t.installCell(cell, rect)
//				where = where.Sub(rect.Point)
//				target := cell.PanelAt(where)
//				t.Tooltip = nil
//				t.TooltipImmediate = false
//				for target != t.AsPanel() {
//					avoid = target.RectToRoot(target.ContentRect(true)).Align()
//					if target.UpdateTooltipCallback != nil {
//						toolbox.Call(func() { avoid = target.UpdateTooltipCallback(cell.PointTo(where, target), avoid) })
//					}
//					if target.Tooltip != nil {
//						t.Tooltip = target.Tooltip
//						t.TooltipImmediate = target.TooltipImmediate
//						break
//					}
//					target = target.parent
//				}
//				t.uninstallCell(cell)
//				return avoid
//			}
//			if cell.Tooltip != nil {
//				t.Tooltip = cell.Tooltip
//				t.TooltipImmediate = cell.TooltipImmediate
//				return t.RectToRoot(t.CellFrame(row, col)).Align()
//			}
//		}
//	}
//	t.Tooltip = nil
//	return Rect{}
//}
//
//// DefaultMouseEnter provides the default mouse enter handling.
//func (t *Table[T]) DefaultMouseEnter(where Point, mod Modifiers) bool {
//	row := t.OverRow(where.Y)
//	col := t.OverColumn(where.X)
//	if t.lastMouseMotionRow != row || t.lastMouseMotionColumn != col {
//		t.DefaultMouseExit()
//		t.lastMouseMotionRow = row
//		t.lastMouseMotionColumn = col
//	}
//	if row != -1 && col != -1 {
//		cell := t.cell(row, col)
//		rect := t.CellFrame(row, col)
//		t.installCell(cell, rect)
//		where = where.Sub(rect.Point)
//		target := cell.PanelAt(where)
//		if target != t.lastMouseEnterCellPanel && t.lastMouseEnterCellPanel != nil {
//			t.DefaultMouseExit()
//			t.lastMouseMotionRow = row
//			t.lastMouseMotionColumn = col
//		}
//		if target.MouseEnterCallback != nil {
//			toolbox.Call(func() { target.MouseEnterCallback(cell.PointTo(where, target), mod) })
//		}
//		t.uninstallCell(cell)
//		t.lastMouseEnterCellPanel = target
//	}
//	return true
//}
//
//// DefaultMouseMove provides the default mouse move handling.
//func (t *Table[T]) DefaultMouseMove(where Point, mod Modifiers) bool {
//	t.DefaultMouseEnter(where, mod)
//	if t.lastMouseEnterCellPanel != nil {
//		row := t.OverRow(where.Y)
//		col := t.OverColumn(where.X)
//		cell := t.cell(row, col)
//		rect := t.CellFrame(row, col)
//		t.installCell(cell, rect)
//		where = where.Sub(rect.Point)
//		if target := cell.PanelAt(where); target.MouseMoveCallback != nil {
//			toolbox.Call(func() { target.MouseMoveCallback(cell.PointTo(where, target), mod) })
//		}
//		t.uninstallCell(cell)
//	}
//	return true
//}
//
//// DefaultMouseExit provides the default mouse exit handling.
//func (t *Table[T]) DefaultMouseExit() bool {
//	if t.lastMouseEnterCellPanel != nil && t.lastMouseEnterCellPanel.MouseExitCallback != nil &&
//		t.lastMouseMotionColumn != -1 && t.lastMouseMotionRow >= 0 && t.lastMouseMotionRow < len(t.rowCache) {
//		cell := t.cell(t.lastMouseMotionRow, t.lastMouseMotionColumn)
//		rect := t.CellFrame(t.lastMouseMotionRow, t.lastMouseMotionColumn)
//		t.installCell(cell, rect)
//		toolbox.Call(func() { t.lastMouseEnterCellPanel.MouseExitCallback() })
//		t.uninstallCell(cell)
//	}
//	t.lastMouseEnterCellPanel = nil
//	t.lastMouseMotionRow = -1
//	t.lastMouseMotionColumn = -1
//	return true
//}
//
//// DefaultMouseDown provides the default mouse down handling.
//func (t *Table[T]) DefaultMouseDown(where Point, button, clickCount int, mod Modifiers) bool {
//	if t.Window().InDrag() {
//		return false
//	}
//	t.RequestFocus()
//	t.wasDragged = false
//	t.dividerDrag = false
//	t.lastSel = zeroUUID
//
//	t.interactionRow = -1
//	t.interactionColumn = -1
//	if button == ButtonLeft {
//		if !t.PreventUserColumnResize {
//			if over := t.OverColumnDivider(where.X); over != -1 {
//				if t.Columns[over].Minimum <= 0 || t.Columns[over].Minimum < t.Columns[over].Maximum {
//					if clickCount == 2 {
//						t.SizeColumnToFit(over, true)
//						t.MarkForRedraw()
//						t.Window().UpdateCursorNow()
//						return true
//					}
//					t.interactionColumn = over
//					t.columnResizeStart = where.X
//					t.columnResizeBase = t.Columns[over].Current
//					t.columnResizeOverhead = t.Padding.Left + t.Padding.Right
//					if t.Columns[over].ID == t.HierarchyColumnID {
//						depth := 0
//						for _, cache := range t.rowCache {
//							if depth < cache.depth {
//								depth = cache.depth
//							}
//						}
//						t.columnResizeOverhead += t.Padding.Left + t.HierarchyIndent*float32(depth+1)
//					}
//					return true
//				}
//			}
//		}
//		for _, one := range t.hitRects {
//			if where.In(one.Rect) {
//				return true
//			}
//		}
//	}
//	if row := t.OverRow(where.Y); row != -1 {
//		if col := t.OverColumn(where.X); col != -1 {
//			cell := t.cell(row, col)
//			if cell.HasInSelfOrDescendants(func(p *Panel) bool { return p.MouseDownCallback != nil }) {
//				t.interactionRow = row
//				t.interactionColumn = col
//				rect := t.CellFrame(row, col)
//				t.installCell(cell, rect)
//				where = where.Sub(rect.Point)
//				stop := false
//				if target := cell.PanelAt(where); target.MouseDownCallback != nil {
//					t.lastMouseDownCellPanel = target
//					toolbox.Call(func() {
//						stop = target.MouseDownCallback(cell.PointTo(where, target), button,
//							clickCount, mod)
//					})
//				}
//				t.uninstallCell(cell)
//				if stop {
//					return stop
//				}
//			}
//		}
//		rowData := t.rowCache[row].row
//		id := rowData.UUID()
//		switch {
//		case mod&ShiftModifier != 0: // Extend selection from anchor
//			selAnchorIndex := -1
//			if t.selAnchor != zeroUUID {
//				for i, c := range t.rowCache {
//					if c.row.UUID() == t.selAnchor {
//						selAnchorIndex = i
//						break
//					}
//				}
//			}
//			if selAnchorIndex != -1 {
//				last := max(selAnchorIndex, row)
//				for i := min(selAnchorIndex, row); i <= last; i++ {
//					t.selMap[t.rowCache[i].row.UUID()] = true
//				}
//				t.notifyOfSelectionChange()
//			} else if !t.selMap[id] { // No anchor, so behave like a regular click
//				t.selMap = make(map[uuid.UUID]bool)
//				t.selMap[id] = true
//				t.selAnchor = id
//				t.notifyOfSelectionChange()
//			}
//		case mod.DiscontiguousSelectionDown(): // Toggle single row
//			if t.selMap[id] {
//				delete(t.selMap, id)
//			} else {
//				t.selMap[id] = true
//			}
//			t.notifyOfSelectionChange()
//		case t.selMap[id]: // Sets lastClick so that on mouse up, we can treat a click and click and hold differently
//			t.lastSel = id
//		default: // If not already selected, replace selection with current row and make it the anchor
//			t.selMap = make(map[uuid.UUID]bool)
//			t.selMap[id] = true
//			t.selAnchor = id
//			t.notifyOfSelectionChange()
//		}
//		t.MarkForRedraw()
//		if button == ButtonLeft && clickCount == 2 && t.DoubleClickCallback != nil && len(t.selMap) != 0 {
//			toolbox.Call(t.DoubleClickCallback)
//		}
//	}
//	return true
//}
//
//func (t *Table[T]) notifyOfSelectionChange() {
//	if t.SelectionChangedCallback != nil {
//		toolbox.Call(t.SelectionChangedCallback)
//	}
//}
//
//// DefaultMouseDrag provides the default mouse drag handling.
//func (t *Table[T]) DefaultMouseDrag(where Point, button int, mod Modifiers) bool {
//	t.wasDragged = true
//	stop := false
//	if t.interactionColumn != -1 {
//		if t.interactionRow == -1 {
//			if button == ButtonLeft && !t.PreventUserColumnResize {
//				width := t.columnResizeBase + where.X - t.columnResizeStart
//				if width < t.columnResizeOverhead {
//					width = t.columnResizeOverhead
//				}
//				minimum := t.Columns[t.interactionColumn].Minimum
//				if minimum > 0 && width < minimum+t.columnResizeOverhead {
//					width = minimum + t.columnResizeOverhead
//				} else {
//					maximum := t.Columns[t.interactionColumn].Maximum
//					if maximum > 0 && width > maximum+t.columnResizeOverhead {
//						width = maximum + t.columnResizeOverhead
//					}
//				}
//				if t.Columns[t.interactionColumn].Current != width {
//					t.Columns[t.interactionColumn].Current = width
//					t.EventuallySyncToModel()
//					t.MarkForRedraw()
//					t.dividerDrag = true
//				}
//				stop = true
//			}
//		} else if t.lastMouseDownCellPanel != nil && t.lastMouseDownCellPanel.MouseDragCallback != nil {
//			cell := t.cell(t.interactionRow, t.interactionColumn)
//			rect := t.CellFrame(t.interactionRow, t.interactionColumn)
//			t.installCell(cell, rect)
//			where = where.Sub(rect.Point)
//			toolbox.Call(func() {
//				stop = t.lastMouseDownCellPanel.MouseDragCallback(cell.PointTo(where, t.lastMouseDownCellPanel), button, mod)
//			})
//			t.uninstallCell(cell)
//		}
//	}
//	return stop
//}
//
//// DefaultMouseUp provides the default mouse up handling.
//func (t *Table[T]) DefaultMouseUp(where Point, button int, mod Modifiers) bool {
//	stop := false
//	if !t.dividerDrag && button == ButtonLeft {
//		for _, one := range t.hitRects {
//			if where.In(one.Rect) {
//				one.handler()
//				stop = true
//				break
//			}
//		}
//	}
//
//	if !t.wasDragged && t.lastSel != zeroUUID {
//		t.ClearSelection()
//		t.selMap[t.lastSel] = true
//		t.selAnchor = t.lastSel
//		t.MarkForRedraw()
//		t.notifyOfSelectionChange()
//	}
//
//	if !stop && t.interactionRow != -1 && t.interactionColumn != -1 && t.lastMouseDownCellPanel != nil &&
//		t.lastMouseDownCellPanel.MouseUpCallback != nil {
//		cell := t.cell(t.interactionRow, t.interactionColumn)
//		rect := t.CellFrame(t.interactionRow, t.interactionColumn)
//		t.installCell(cell, rect)
//		where = where.Sub(rect.Point)
//		toolbox.Call(func() {
//			stop = t.lastMouseDownCellPanel.MouseUpCallback(cell.PointTo(where, t.lastMouseDownCellPanel), button, mod)
//		})
//		t.uninstallCell(cell)
//	}
//	t.lastMouseDownCellPanel = nil
//	return stop
//}
//
//// DefaultKeyDown provides the default key down handling.
//func (t *Table[T]) DefaultKeyDown(keyCode KeyCode, mod Modifiers, _ bool) bool {
//	if IsControlAction(keyCode, mod) {
//		if t.DoubleClickCallback != nil && len(t.selMap) != 0 {
//			toolbox.Call(t.DoubleClickCallback)
//		}
//		return true
//	}
//	switch keyCode {
//	case KeyLeft:
//		if t.HasSelection() {
//			altered := false
//			for _, row := range t.SelectedRows(false) {
//				if row.IsOpen() {
//					row.SetOpen(false)
//					altered = true
//				}
//			}
//			if altered {
//				t.SyncToModel()
//				t.PruneSelectionOfUndisclosedNodes()
//			}
//		}
//	case KeyRight:
//		if t.HasSelection() {
//			altered := false
//			for _, row := range t.SelectedRows(false) {
//				if !row.IsOpen() {
//					row.SetOpen(true)
//					altered = true
//				}
//			}
//			if altered {
//				t.SyncToModel()
//			}
//		}
//	case KeyUp:
//		var i int
//		if t.HasSelection() {
//			i = max(t.FirstSelectedRowIndex()-1, 0)
//		} else {
//			i = len(t.rowCache) - 1
//		}
//		if !mod.ShiftDown() {
//			t.ClearSelection()
//		}
//		t.SelectByIndex(i)
//		t.ScrollRowCellIntoView(i, 0)
//	case KeyDown:
//		i := min(t.LastSelectedRowIndex()+1, len(t.rowCache)-1)
//		if !mod.ShiftDown() {
//			t.ClearSelection()
//		}
//		t.SelectByIndex(i)
//		t.ScrollRowCellIntoView(i, 0)
//	case KeyHome:
//		if mod.ShiftDown() && t.HasSelection() {
//			t.SelectRange(0, t.FirstSelectedRowIndex())
//		} else {
//			t.ClearSelection()
//			t.SelectByIndex(0)
//		}
//		t.ScrollRowCellIntoView(0, 0)
//	case KeyEnd:
//		if mod.ShiftDown() && t.HasSelection() {
//			t.SelectRange(t.LastSelectedRowIndex(), len(t.rowCache)-1)
//		} else {
//			t.ClearSelection()
//			t.SelectByIndex(len(t.rowCache) - 1)
//		}
//		t.ScrollRowCellIntoView(len(t.rowCache)-1, 0)
//	default:
//		return false
//	}
//	return true
//}

// PruneSelectionOfUndisclosedNodes removes any nodes in the selection map that are no longer disclosed from the
// selection map.
//func (t *Table[T]) PruneSelectionOfUndisclosedNodes() {
//	if !t.selNeedsPrune {
//		return
//	}
//	t.selNeedsPrune = false
//	if len(t.selMap) == 0 {
//		return
//	}
//	needsNotify := false
//	selMap := make(map[uuid.UUID]bool, len(t.selMap))
//	for _, entry := range t.rowCache {
//		id := entry.row.UUID()
//		if t.selMap[id] {
//			selMap[id] = true
//		} else {
//			needsNotify = true
//		}
//	}
//	t.selMap = selMap
//	if needsNotify {
//		t.notifyOfSelectionChange()
//	}
//}

// FirstSelectedRowIndex returns the first selected row index, or -1 if there is no selection.
func (t *Table[T]) FirstSelectedRowIndex() int {
	if len(t.selMap) == 0 {
		return -1
	}
	for i, entry := range t.rowCache {
		if t.selMap[entry.row.UUID()] {
			return i
		}
	}
	return -1
}

// LastSelectedRowIndex returns the last selected row index, or -1 if there is no selection.
func (t *Table[T]) LastSelectedRowIndex() int {
	if len(t.selMap) == 0 {
		return -1
	}
	for i := len(t.rowCache) - 1; i >= 0; i-- {
		if t.selMap[t.rowCache[i].row.UUID()] {
			return i
		}
	}
	return -1
}

// IsRowOrAnyParentSelected returns true if the specified row index or any of its parents are selected.
func (t *Table[T]) IsRowOrAnyParentSelected(index int) bool {
	if index < 0 || index >= len(t.rowCache) {
		return false
	}
	for index >= 0 {
		if t.selMap[t.rowCache[index].row.UUID()] {
			return true
		}
		index = t.rowCache[index].parent
	}
	return false
}

// IsRowSelected returns true if the specified row index is selected.
func (t *Table[T]) IsRowSelected(index int) bool {
	if index < 0 || index >= len(t.rowCache) {
		return false
	}
	return t.selMap[t.rowCache[index].row.UUID()]
}

// SelectedRows returns the currently selected rows. If 'minimal' is true, then children of selected rows that may also
// be selected are not returned, just the topmost row that is selected in any given hierarchy.
//func (t *Table[T]) SelectedRows(minimal bool) []T {
//	t.PruneSelectionOfUndisclosedNodes()
//	if len(t.selMap) == 0 {
//		return nil
//	}
//	rows := make([]T, 0, len(t.selMap))
//	for _, entry := range t.rowCache {
//		if t.selMap[entry.row.UUID()] && (!minimal || entry.parent == -1 || !t.IsRowOrAnyParentSelected(entry.parent)) {
//			rows = append(rows, entry.row)
//		}
//	}
//	return rows
//}

// CopySelectionMap returns a copy of the current selection map.
//func (t *Table[T]) CopySelectionMap() map[uuid.UUID]bool {
//	t.PruneSelectionOfUndisclosedNodes()
//	return copySelMap(t.selMap)
//}

// SetSelectionMap sets the current selection map.
//func (t *Table[T]) SetSelectionMap(selMap map[uuid.UUID]bool) {
//	t.selMap = copySelMap(selMap)
//	t.selNeedsPrune = true
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}

func copySelMap(selMap map[uuid.UUID]bool) map[uuid.UUID]bool {
	result := make(map[uuid.UUID]bool, len(selMap))
	for k, v := range selMap {
		result[k] = v
	}
	return result
}

// HasSelection returns true if there is a selection.
//func (t *Table[T]) HasSelection() bool {
//	t.PruneSelectionOfUndisclosedNodes()
//	return len(t.selMap) != 0
//}
//
//// SelectionCount returns the number of rows explicitly selected.
//func (t *Table[T]) SelectionCount() int {
//	t.PruneSelectionOfUndisclosedNodes()
//	return len(t.selMap)
//}
//
//// ClearSelection clears the selection.
//func (t *Table[T]) ClearSelection() {
//	if len(t.selMap) == 0 {
//		return
//	}
//	t.selMap = make(map[uuid.UUID]bool)
//	t.selNeedsPrune = false
//	t.selAnchor = zeroUUID
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}
//
//// SelectAll selects all rows.
//func (t *Table[T]) SelectAll() {
//	t.selMap = make(map[uuid.UUID]bool, len(t.rowCache))
//	t.selNeedsPrune = false
//	t.selAnchor = zeroUUID
//	for _, cache := range t.rowCache {
//		id := cache.row.UUID()
//		t.selMap[id] = true
//		if t.selAnchor == zeroUUID {
//			t.selAnchor = id
//		}
//	}
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}
//
//// SelectByIndex selects the given indexes. The first one will be considered the anchor selection if no existing anchor
//// selection exists.
//func (t *Table[T]) SelectByIndex(indexes ...int) {
//	for _, index := range indexes {
//		if index >= 0 && index < len(t.rowCache) {
//			id := t.rowCache[index].row.UUID()
//			t.selMap[id] = true
//			t.selNeedsPrune = true
//			if t.selAnchor == zeroUUID {
//				t.selAnchor = id
//			}
//		}
//	}
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}
//
//// SelectRange selects the given range. The start will be considered the anchor selection if no existing anchor
//// selection exists.
//func (t *Table[T]) SelectRange(start, end int) {
//	start = max(start, 0)
//	end = min(end, len(t.rowCache)-1)
//	if start > end {
//		return
//	}
//	for i := start; i <= end; i++ {
//		id := t.rowCache[i].row.UUID()
//		t.selMap[id] = true
//		t.selNeedsPrune = true
//		if t.selAnchor == zeroUUID {
//			t.selAnchor = id
//		}
//	}
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}
//
//// DeselectByIndex deselects the given indexes.
//func (t *Table[T]) DeselectByIndex(indexes ...int) {
//	for _, index := range indexes {
//		if index >= 0 && index < len(t.rowCache) {
//			delete(t.selMap, t.rowCache[index].row.UUID())
//		}
//	}
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}
//
//// DeselectRange deselects the given range.
//func (t *Table[T]) DeselectRange(start, end int) {
//	start = max(start, 0)
//	end = min(end, len(t.rowCache)-1)
//	if start > end {
//		return
//	}
//	for i := start; i <= end; i++ {
//		delete(t.selMap, t.rowCache[i].row.UUID())
//	}
//	t.MarkForRedraw()
//	t.notifyOfSelectionChange()
//}

// DiscloseRow ensures the given row can be viewed by opening all parents that lead to it. Returns true if any
// modification was made.
func (t *Table[T]) DiscloseRow(row T, delaySync bool) bool {
	modified := false
	p := row.Parent()
	var zero T
	for p != zero {
		if !p.IsOpen() {
			p.SetOpen(true)
			modified = true
		}
		p = p.Parent()
	}
	if modified {
		//if delaySync {
		//	t.EventuallySyncToModel()
		//} else {
		//	t.SyncToModel()
		//}
	}
	return modified
}

// RootRowCount returns the number of top-level rows.
func (t *Table[T]) RootRowCount() int {
	if t.filteredRows != nil {
		return len(t.filteredRows)
	}
	return t.Model.RootRowCount()
}

// RootRows returns the top-level rows. Do not alter the returned list.
func (t *Table[T]) RootRows() []T {
	if t.filteredRows != nil {
		return t.filteredRows
	}
	return t.Model.RootRows()
}

// SetRootRows sets the top-level rows this table will display. This will call SyncToModel() automatically.
func (t *Table[T]) SetRootRows(rows []T) {
	t.filteredRows = nil
	t.Model.SetRootRows(rows)
	t.selMap = make(map[uuid.UUID]bool)
	t.selNeedsPrune = false
	t.selAnchor = zeroUUID
	//t.SyncToModel()
}

// SyncToModel causes the table to update its internal caches to reflect the current model.
//func (t *Table[T]) SyncToModel() {
//	rowCount := 0
//	roots := t.RootRows()
//	if t.filteredRows != nil {
//		rowCount = len(t.filteredRows)
//	} else {
//		for _, row := range roots {
//			rowCount += t.countOpenRowChildrenRecursively(row)
//		}
//	}
//	t.rowCache = make([]tableCache[T], rowCount)
//	j := 0
//	for _, row := range roots {
//		j = t.buildRowCacheEntry(row, -1, j, 0)
//	}
//	t.selNeedsPrune = true
//	_, pref, _ := t.DefaultSizes(Size{})
//	rect := t.FrameRect()
//	rect.Size = pref
//	t.SetFrameRect(rect)
//	t.MarkForRedraw()
//	t.MarkForLayoutRecursivelyUpward()
//}

func (t *Table[T]) countOpenRowChildrenRecursively(row T) int {
	count := 1
	if row.CanHaveChildren() && row.IsOpen() {
		for _, child := range row.Children() {
			count += t.countOpenRowChildrenRecursively(child)
		}
	}
	return count
}

func (t *Table[T]) buildRowCacheEntry(row T, parentIndex, index, depth int) int {
	t.rowCache[index].row = row
	t.rowCache[index].parent = parentIndex
	t.rowCache[index].depth = depth
	//t.rowCache[index].height = t.heightForColumns(row, index, depth)
	parentIndex = index
	index++
	if t.filteredRows == nil && row.CanHaveChildren() && row.IsOpen() {
		for _, child := range row.Children() {
			index = t.buildRowCacheEntry(child, parentIndex, index, depth+1)
		}
	}
	return index
}

//func (t *Table[T]) heightForColumns(rowData T, row, depth int) float32 {
//	var height float32
//	for col := range t.Columns {
//		w := t.Columns[col].Current
//		if w <= 0 {
//			continue
//		}
//		w -= t.Padding.Left + t.Padding.Right
//		if t.Columns[col].ID == t.HierarchyColumnID {
//			w -= t.Padding.Left + t.HierarchyIndent*float32(depth+1)
//		}
//		size := t.cellPrefSize(rowData, row, col, w)
//		size.Height += t.Padding.Top + t.Padding.Bottom
//		if height < size.Height {
//			height = size.Height
//		}
//	}
//	return max(xmath.Ceil(height), t.MinimumRowHeight)
//}
//
//func (t *Table[T]) cellPrefSize(rowData T, row, col int, widthConstraint float32) Size {
//	fg, bg, selected, indirectlySelected, focused := t.cellParams(row, col)
//	cell := rowData.ColumnCell(row, col, fg, bg, selected, indirectlySelected, focused).AsPanel()
//	_, size, _ := cell.Sizes(Size{Width: widthConstraint})
//	return size
//}
//
//// SizeColumnsToFitWithExcessIn sizes each column to its preferred size, with the exception of the column with the given
//// ID, which gets set to any remaining width left over. If the provided column ID doesn't exist, the first column will
//// be used instead.
//func (t *Table[T]) SizeColumnsToFitWithExcessIn(columnID int) {
//	excessColumnIndex := max(t.ColumnIndexForID(columnID), 0)
//	current := make([]float32, len(t.Columns))
//	for col := range t.Columns {
//		current[col] = max(t.Columns[col].Minimum, 0)
//		t.Columns[col].Current = 0
//	}
//	for row, cache := range t.rowCache {
//		for col := range t.Columns {
//			if col == excessColumnIndex {
//				continue
//			}
//			pref := t.cellPrefSize(cache.row, row, col, 0)
//			minimum := t.Columns[col].AutoMinimum
//			if minimum > 0 && pref.Width < minimum {
//				pref.Width = minimum
//			} else {
//				maximum := t.Columns[col].AutoMaximum
//				if maximum > 0 && pref.Width > maximum {
//					pref.Width = maximum
//				}
//			}
//			pref.Width += t.Padding.Left + t.Padding.Right
//			if t.Columns[col].ID == t.HierarchyColumnID {
//				pref.Width += t.Padding.Left + t.HierarchyIndent*float32(cache.depth+1)
//			}
//			if current[col] < pref.Width {
//				current[col] = pref.Width
//			}
//		}
//	}
//	width := t.ContentRect(false).Width
//	if t.ShowColumnDivider {
//		width -= float32(len(t.Columns) - 1)
//	}
//	for col := range current {
//		if col == excessColumnIndex {
//			continue
//		}
//		t.Columns[col].Current = current[col]
//		width -= current[col]
//	}
//	t.Columns[excessColumnIndex].Current = max(width, t.Columns[excessColumnIndex].Minimum)
//	for row, cache := range t.rowCache {
//		t.rowCache[row].height = t.heightForColumns(cache.row, row, cache.depth)
//	}
//}
//
//// SizeColumnsToFit sizes each column to its preferred size. If 'adjust' is true, the Table's FrameRect will be set to
//// its preferred size as well.
//func (t *Table[T]) SizeColumnsToFit(adjust bool) {
//	current := make([]float32, len(t.Columns))
//	for col := range t.Columns {
//		current[col] = max(t.Columns[col].Minimum, 0)
//		t.Columns[col].Current = 0
//	}
//	for row, cache := range t.rowCache {
//		for col := range t.Columns {
//			pref := t.cellPrefSize(cache.row, row, col, 0)
//			minimum := t.Columns[col].AutoMinimum
//			if minimum > 0 && pref.Width < minimum {
//				pref.Width = minimum
//			} else {
//				maximum := t.Columns[col].AutoMaximum
//				if maximum > 0 && pref.Width > maximum {
//					pref.Width = maximum
//				}
//			}
//			pref.Width += t.Padding.Left + t.Padding.Right
//			if t.Columns[col].ID == t.HierarchyColumnID {
//				pref.Width += t.Padding.Left + t.HierarchyIndent*float32(cache.depth+1)
//			}
//			if current[col] < pref.Width {
//				current[col] = pref.Width
//			}
//		}
//	}
//	for col := range current {
//		t.Columns[col].Current = current[col]
//	}
//	for row, cache := range t.rowCache {
//		t.rowCache[row].height = t.heightForColumns(cache.row, row, cache.depth)
//	}
//	if adjust {
//		_, pref, _ := t.DefaultSizes(Size{})
//		rect := t.FrameRect()
//		rect.Size = pref
//		t.SetFrameRect(rect)
//	}
//}
//
//// SizeColumnToFit sizes the specified column to its preferred size. If 'adjust' is true, the Table's FrameRect will be
//// set to its preferred size as well.
//func (t *Table[T]) SizeColumnToFit(col int, adjust bool) {
//	if col < 0 || col >= len(t.Columns) {
//		return
//	}
//	current := max(t.Columns[col].Minimum, 0)
//	t.Columns[col].Current = 0
//	for row, cache := range t.rowCache {
//		pref := t.cellPrefSize(cache.row, row, col, 0)
//		minimum := t.Columns[col].AutoMinimum
//		if minimum > 0 && pref.Width < minimum {
//			pref.Width = minimum
//		} else {
//			maximum := t.Columns[col].AutoMaximum
//			if maximum > 0 && pref.Width > maximum {
//				pref.Width = maximum
//			}
//		}
//		pref.Width += t.Padding.Left + t.Padding.Right
//		if t.Columns[col].ID == t.HierarchyColumnID {
//			pref.Width += t.Padding.Left + t.HierarchyIndent*float32(cache.depth+1)
//		}
//		if current < pref.Width {
//			current = pref.Width
//		}
//	}
//	t.Columns[col].Current = current
//	for row, cache := range t.rowCache {
//		t.rowCache[row].height = t.heightForColumns(cache.row, row, cache.depth)
//	}
//	if adjust {
//		_, pref, _ := t.DefaultSizes(Size{})
//		rect := t.FrameRect()
//		rect.Size = pref
//		t.SetFrameRect(rect)
//	}
//}
//
//// EventuallySizeColumnsToFit sizes each column to its preferred size after a short delay, allowing multiple
//// back-to-back calls to this function to only do work once. If 'adjust' is true, the Table's FrameRect will be set to
//// its preferred size as well.
//func (t *Table[T]) EventuallySizeColumnsToFit(adjust bool) {
//	if !t.awaitingSizeColumnsToFit {
//		t.awaitingSizeColumnsToFit = true
//		InvokeTaskAfter(func() {
//			t.SizeColumnsToFit(adjust)
//			t.awaitingSizeColumnsToFit = false
//		}, 20*time.Millisecond)
//	}
//}
//
//// EventuallySyncToModel syncs the table to its underlying model after a short delay, allowing multiple back-to-back
//// calls to this function to only do work once.
//func (t *Table[T]) EventuallySyncToModel() {
//	if !t.awaitingSyncToModel {
//		t.awaitingSyncToModel = true
//		InvokeTaskAfter(func() {
//			t.SyncToModel()
//			t.awaitingSyncToModel = false
//		}, 20*time.Millisecond)
//	}
//}
//
//// DefaultSizes provides the default sizing.
//func (t *Table[T]) DefaultSizes(_ Size) (minSize, prefSize, maxSize Size) {
//	for col := range t.Columns {
//		prefSize.Width += t.Columns[col].Current
//	}
//	startRow, endBeforeRow := t.CurrentDrawRowRange()
//	for _, cache := range t.rowCache[startRow:endBeforeRow] {
//		prefSize.Height += cache.height
//	}
//	if t.ShowColumnDivider {
//		prefSize.Width += float32(len(t.Columns) - 1)
//	}
//	if t.ShowRowDivider {
//		prefSize.Height += float32((endBeforeRow - startRow) - 1)
//	}
//	if border := t.Border(); border != nil {
//		prefSize = prefSize.Add(border.Insets().Size())
//	}
//	prefSize = prefSize.Ceil()
//	return prefSize, prefSize, prefSize
//}

// RowFromIndex returns the row data for the given index.
func (t *Table[T]) RowFromIndex(index int) T {
	if index < 0 || index >= len(t.rowCache) {
		var zero T
		return zero
	}
	return t.rowCache[index].row
}

// RowToIndex returns the row's index within the displayed data, or -1 if it isn't currently in the disclosed rows.
func (t *Table[T]) RowToIndex(rowData T) int {
	id := rowData.UUID()
	for row, data := range t.rowCache {
		if data.row.UUID() == id {
			return row
		}
	}
	return -1
}

// LastRowIndex returns the index of the last row. Will be -1 if there are no rows.
func (t *Table[T]) LastRowIndex() int {
	return len(t.rowCache) - 1
}

func (t *Table[T]) ScrollRowIntoView(row int) {
	if frame := t.RowFrame(row); !frame.Empty() {
		t.ScrollRectIntoView(frame)
	}
}

func (t *Table[T]) ScrollRowCellIntoView(row, col int) {
	if frame := t.CellFrame(row, col); !frame.Empty() {
		t.ScrollRectIntoView(frame)
	}
}

package demo

/*

// FindRowIndexByID returns the row index of the row with the given ID in the given table.
func FindRowIndexByID[T gurps.NodeTypes](table *unison.Table[*Node[T]], id uuid.UUID) int {
	_, i := rowIndex(id, 0, table.RootRows())
	return i
}

func rowIndex[T gurps.NodeTypes](id uuid.UUID, startIndex int, rows []*Node[T]) (updatedStartIndex, result int) {
	for _, row := range rows {
		if id == row.dataAsNode.UUID() {
			return 0, startIndex
		}
		startIndex++
		if row.IsOpen() {
			if startIndex, result = rowIndex(id, startIndex, row.Children()); result != -1 {
				return 0, result
			}
		}
	}
	return startIndex, -1
}

// InsertItems into a table.
func InsertItems[T gurps.NodeTypes](owner Rebuildable, table *unison.Table[*Node[T]], topList func() []T, setTopList func([]T), rowData func(table *unison.Table[*Node[T]]) []*Node[T], items ...T) {
	if len(items) == 0 {
		return
	}
	var undo *unison.UndoEdit[*TableUndoEditData[T]]
	mgr := unison.UndoManagerFor(table)
	if mgr != nil {
		undo = &unison.UndoEdit[*TableUndoEditData[T]]{
			ID:         unison.NextUndoID(),
			EditName:   fmt.Sprintf(i18n.Text("Insert %s"), gurps.AsNode(items[0]).Kind()),
			UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
			RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
			AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
			BeforeData: NewTableUndoEditData(table),
		}
	}
	var target, zero T
	i := table.FirstSelectedRowIndex()
	if i != -1 {
		row := table.RowFromIndex(i)
		if target = row.Data(); target != zero {
			if row.CanHaveChildren() {
				// Target is container, append to end of that container
				SetParents(items, target)
				row.dataAsNode.SetChildren(append(row.dataAsNode.NodeChildren(), items...))
			} else {
				// Target isn't a container. If it has a parent, insert after the target within that parent.
				parent := row.Parent()
				if parentData := parent.Data(); parentData != zero {
					SetParents(items, parentData)
					children := parent.dataAsNode.NodeChildren()
					parent.dataAsNode.SetChildren(slices.Insert(children, slices.Index(children, target)+1, items...))
				} else {
					// Otherwise, insert after the target within the top-level list.
					SetParents(items, zero)
					list := topList()
					setTopList(slices.Insert(list, slices.Index(list, target)+1, items...))
				}
			}
		}
	}
	if target == zero {
		// There was no selection, so append to the end of the top-level list.
		SetParents(items, zero)
		setTopList(append(topList(), items...))
	}
	MarkModified(table)
	table.SetRootRows(rowData(table))
	table.ValidateScrollRoot()
	table.RequestFocus()
	selMap := make(map[uuid.UUID]bool)
	for _, item := range items {
		selMap[gurps.AsNode(item).UUID()] = true
	}
	table.SetSelectionMap(selMap)
	table.ScrollRowCellIntoView(table.LastSelectedRowIndex(), 0)
	table.ScrollRowCellIntoView(table.FirstSelectedRowIndex(), 0)
	if mgr != nil && undo != nil {
		undo.AfterData = NewTableUndoEditData(table)
		mgr.Add(undo)
	}
	owner.Rebuild(true)
}

// SetParents of each item.
func SetParents[T gurps.NodeTypes](items []T, parent T) {
	for _, item := range items {
		gurps.AsNode(item).SetParent(parent)
	}
}

// ExtractNodeDataFromList returns the underlying node data.
func ExtractNodeDataFromList[T gurps.NodeTypes](list []*Node[T]) []T {
	dataList := make([]T, 0, len(list))
	for _, child := range list {
		dataList = append(dataList, child.data)
	}
	return dataList
}



const containerMarker = "\000"

// ItemVariant holds the type of item variant to create.
type ItemVariant int

// Possible values for ItemVariant.
const (
	NoItemVariant ItemVariant = iota
	ContainerItemVariant
	AlternateItemVariant
)

// TableProvider defines the methods a table provider must contain.
type TableProvider[T gurps.NodeTypes] interface {
	unison.TableModel[*Node[T]]
	gurps.EntityProvider
	SetTable(table *unison.Table[*Node[T]])
	RootData() []T
	SetRootData(data []T)
	DragKey() string
	DragSVG() *unison.SVG
	DropShouldMoveData(from, to *unison.Table[*Node[T]]) bool
	ProcessDropData(from, to *unison.Table[*Node[T]])
	AltDropSupport() *AltDropSupport
	ItemNames() (singular, plural string)
	Headers() []unison.TableColumnHeader[*Node[T]]
	SyncHeader(headers []unison.TableColumnHeader[*Node[T]])
	ColumnIDs() []int
	HierarchyColumnID() int
	ExcessWidthColumnID() int
	ContextMenuItems() []ContextMenuItem
	OpenEditor(owner Rebuildable, table *unison.Table[*Node[T]])
	CreateItem(owner Rebuildable, table *unison.Table[*Node[T]], variant ItemVariant)
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	RefKey() string
	AllTags() []string
}

// NewNodeTable creates a new node table of the specified type, returning the header and table. Pass nil for 'font' if
// this should be a standalone top-level table for a dockable. Otherwise, pass in the typical font used for a cell.
func NewNodeTable[T gurps.NodeTypes](provider TableProvider[T], font unison.Font) (header *unison.TableHeader[*Node[T]], table *unison.Table[*Node[T]]) {
	table = unison.NewTable[*Node[T]](provider)
	provider.SetTable(table)
	table.HierarchyColumnID = provider.HierarchyColumnID()
	layoutData := &unison.FlexLayoutData{
		HAlign: align.Fill,
		VAlign: align.Fill,
		HGrab:  true,
		VGrab:  true,
	}
	if font != nil {
		table.Padding.Top = 0
		table.Padding.Bottom = 0
		table.HierarchyIndent = font.LineHeight()
		table.MinimumRowHeight = font.LineHeight()
		layoutData.MinSize = unison.Size{Height: 4 + gurps.PageFieldPrimaryFont.LineHeight()}
	}
	table.SetLayoutData(layoutData)

	ids := provider.ColumnIDs()
	headers := provider.Headers()
	table.Columns = make([]unison.ColumnInfo, len(headers))
	for i := range table.Columns {
		_, pref, _ := headers[i].AsPanel().Sizes(unison.Size{})
		pref.Width += table.Padding.Left + table.Padding.Right
		table.Columns[i].ID = ids[i]
		table.Columns[i].AutoMinimum = pref.Width
		table.Columns[i].AutoMaximum = max(float32(gurps.GlobalSettings().General.MaximumAutoColWidth), pref.Width)
		table.Columns[i].Minimum = pref.Width
		table.Columns[i].Maximum = 10000
	}
	header = unison.NewTableHeader(table, headers...)
	header.Less = flexibleLess
	header.BackgroundInk = gurps.HeaderColor
	header.SetBorder(header.HeaderBorder)
	header.SetLayoutData(&unison.FlexLayoutData{
		HAlign: align.Fill,
		VAlign: align.Fill,
		HGrab:  true,
	})

	table.DoubleClickCallback = func() { table.PerformCmd(nil, OpenEditorItemID) }
	table.KeyDownCallback = func(keyCode unison.KeyCode, mod unison.Modifiers, repeat bool) bool {
		if mod == 0 && (keyCode == unison.KeyBackspace || keyCode == unison.KeyDelete) {
			table.PerformCmd(table, unison.DeleteItemID)
			return true
		}
		return table.DefaultKeyDown(keyCode, mod, repeat)
	}
	singular, plural := provider.ItemNames()
	table.InstallDragSupport(provider.DragSVG(), provider.DragKey(), singular, plural)
	if font != nil {
		table.FrameChangeCallback = func() {
			table.SizeColumnsToFitWithExcessIn(provider.ExcessWidthColumnID())
		}
	}

	table.MouseDownCallback = func(where unison.Point, button, clickCount int, mod unison.Modifiers) bool {
		stop := table.DefaultMouseDown(where, button, clickCount, mod)
		if button == unison.ButtonRight && clickCount == 1 && !table.Window().InDrag() {
			f := unison.DefaultMenuFactory()
			cm := f.NewMenu(unison.PopupMenuTemporaryBaseID|unison.ContextMenuIDFlag, "", nil)
			id := 1
			for _, one := range provider.ContextMenuItems() {
				if one.ID == -1 {
					cm.InsertSeparator(-1, true)
				} else {
					InsertCmdContextMenuItem(table, one.Title, one.ID, &id, cm)
				}
			}
			count := cm.Count()
			if count > 0 {
				count--
				if cm.ItemAtIndex(count).IsSeparator() {
					cm.RemoveItem(count)
				}
				table.FlushDrawing()
				cm.Popup(unison.Rect{
					Point: table.PointToRoot(where),
					Size: unison.Size{
						Width:  1,
						Height: 1,
					},
				}, 0)
			}
			cm.Dispose()
		}
		return stop
	}

	table.InstallCmdHandlers(CopyToSheetItemID, func(_ any) bool { return canCopySelectionToSheet(table) },
		func(_ any) { copySelectionToSheet(table) })
	table.InstallCmdHandlers(CopyToTemplateItemID, func(_ any) bool { return canCopySelectionToTemplate(table) },
		func(_ any) { copySelectionToTemplate(table) })
	if t, ok := (any(table)).(*unison.Table[*Node[*gurps.Equipment]]); ok {
		t.InstallCmdHandlers(IncrementItemID,
			func(_ any) bool { return canAdjustQuantity(t, true) },
			func(_ any) { adjustQuantity(unison.AncestorOrSelf[Rebuildable](t), t, true) })
		t.InstallCmdHandlers(DecrementItemID,
			func(_ any) bool { return canAdjustQuantity(t, false) },
			func(_ any) { adjustQuantity(unison.AncestorOrSelf[Rebuildable](t), t, false) })
		t.InstallCmdHandlers(IncrementUsesItemID,
			func(_ any) bool { return canAdjustUses(t, 1) },
			func(_ any) { adjustUses(unison.AncestorOrSelf[Rebuildable](t), t, 1) })
		t.InstallCmdHandlers(DecrementUsesItemID,
			func(_ any) bool { return canAdjustUses(t, -1) },
			func(_ any) { adjustUses(unison.AncestorOrSelf[Rebuildable](t), t, -1) })
	}

	return header, table
}

func isAcceptableTypeForSheetOrTemplate(data any) bool {
	switch data.(type) {
	case *gurps.Equipment, *gurps.Note, *gurps.Skill, *gurps.Spell, *gurps.Trait:
		return true
	default:
		return false
	}
}

func canCopySelectionToSheet[T gurps.NodeTypes](table *unison.Table[*Node[T]]) bool {
	var t T
	return table.HasSelection() && len(OpenSheets(unison.Ancestor[*Sheet](table))) > 0 && isAcceptableTypeForSheetOrTemplate(t)
}

func canCopySelectionToTemplate[T gurps.NodeTypes](table *unison.Table[*Node[T]]) bool {
	var t T
	return table.HasSelection() && len(OpenTemplates(unison.Ancestor[*Template](table))) > 0 && isAcceptableTypeForSheetOrTemplate(t)
}

func copySelectionToSheet[T gurps.NodeTypes](table *unison.Table[*Node[T]]) {
	if table.HasSelection() {
		if sheets := PromptForDestination(OpenSheets(unison.Ancestor[*Sheet](table))); len(sheets) > 0 {
			sel := table.SelectedRows(true)
			for _, s := range sheets {
				var targetTable *unison.Table[*Node[T]]
				var postProcessor func(rows []*Node[T])
				switch any(sel[0].Data()).(type) {
				case *gurps.Trait:
					targetTable = convertTable[T](s.Traits.Table)
					postProcessor = func(rows []*Node[T]) {
						s.Traits.provider.ProcessDropData(nil, s.Traits.Table)
					}
				case *gurps.Skill:
					targetTable = convertTable[T](s.Skills.Table)
					postProcessor = func(rows []*Node[T]) {
						s.Skills.provider.ProcessDropData(nil, s.Skills.Table)
					}
				case *gurps.Spell:
					targetTable = convertTable[T](s.Spells.Table)
					postProcessor = func(rows []*Node[T]) {
						s.Spells.provider.ProcessDropData(nil, s.Spells.Table)
					}
				case *gurps.Equipment:
					targetTable = convertTable[T](s.CarriedEquipment.Table)
					postProcessor = func(rows []*Node[T]) {
						s.CarriedEquipment.provider.ProcessDropData(nil, s.CarriedEquipment.Table)
					}
				case *gurps.Note:
					targetTable = convertTable[T](s.Notes.Table)
					postProcessor = func(rows []*Node[T]) {
						s.Notes.provider.ProcessDropData(nil, s.Notes.Table)
					}
				default:
					continue
				}
				if targetTable != nil {
					CopyRowsTo(targetTable, sel, postProcessor, true)
					ProcessModifiersForSelection(targetTable)
					ProcessNameablesForSelection(targetTable)
				}
			}
		}
	}
}

func copySelectionToTemplate[T gurps.NodeTypes](table *unison.Table[*Node[T]]) {
	if table.HasSelection() {
		if templates := PromptForDestination(OpenTemplates(unison.Ancestor[*Template](table))); len(templates) > 0 {
			sel := table.SelectedRows(true)
			for _, t := range templates {
				switch any(sel[0].Data()).(type) {
				case *gurps.Trait:
					CopyRowsTo(convertTable[T](t.Traits.Table), sel, nil, true)
				case *gurps.Skill:
					CopyRowsTo(convertTable[T](t.Skills.Table), sel, nil, true)
				case *gurps.Spell:
					CopyRowsTo(convertTable[T](t.Spells.Table), sel, nil, true)
				case *gurps.Equipment:
					CopyRowsTo(convertTable[T](t.Equipment.Table), sel, nil, true)
				case *gurps.Note:
					CopyRowsTo(convertTable[T](t.Notes.Table), sel, nil, true)
				}
			}
		}
	}
}

func convertTable[T gurps.NodeTypes](table any) *unison.Table[*Node[T]] {
	// This is here just to get around limitations in the way Go generics behave
	if t, ok := table.(*unison.Table[*Node[T]]); ok {
		return t
	}
	return nil
}

// InsertCmdContextMenuItem inserts a context menu item for the given command.
func InsertCmdContextMenuItem[T gurps.NodeTypes](table *unison.Table[*Node[T]], title string, cmdID int, id *int, cm unison.Menu) {
	if table.CanPerformCmd(table, cmdID) {
		useID := *id
		*id++
		cm.InsertItem(-1, cm.Factory().NewItem(unison.PopupMenuTemporaryBaseID+useID, title, unison.KeyBinding{}, nil,
			func(item unison.MenuItem) {
				table.PerformCmd(table, cmdID)
			}))
	}
}

func flexibleLess(s1, s2 string) bool {
	c1 := strings.HasPrefix(s1, containerMarker)
	c2 := strings.HasPrefix(s2, containerMarker)
	if c1 != c2 {
		return c1
	}
	if c1 {
		s1 = s1[1:]
	}
	if c2 {
		s2 = s2[1:]
	}
	if n1, err := fxp.FromString(s1); err == nil {
		var n2 fxp.Int
		if n2, err = fxp.FromString(s2); err == nil {
			return n1 < n2
		}
	}
	return txt.NaturalLess(s1, s2, true)
}

// OpenEditor opens an editor for each selected row in the table.
func OpenEditor[T gurps.NodeTypes](table *unison.Table[*Node[T]], edit func(item T)) {
	var zero T
	selection := table.SelectedRows(false)
	if len(selection) > 4 {
		if unison.QuestionDialog(i18n.Text("Are you sure you want to open all of these?"),
			fmt.Sprintf(i18n.Text("%d editors will be opened."), len(selection))) != unison.ModalResponseOK {
			return
		}
	}
	for _, row := range selection {
		if data := row.Data(); data != zero {
			edit(data)
		}
	}
}

// DeleteSelection removes the selected nodes from the table.
func DeleteSelection[T gurps.NodeTypes](table *unison.Table[*Node[T]], recordUndo bool) {
	if provider, ok := any(table.Model).(TableProvider[T]); ok && !table.IsFiltered() && table.HasSelection() {
		sel := table.SelectedRows(true)
		ids := make(map[uuid.UUID]bool, len(sel))
		list := make([]T, 0, len(sel))
		var zero T
		for _, row := range sel {
			unison.CollectUUIDsFromRow(row, ids)
			if target := row.Data(); target != zero {
				list = append(list, target)
			}
		}
		if !CloseUUID(ids) {
			return
		}
		var undo *unison.UndoEdit[*TableUndoEditData[T]]
		var mgr *unison.UndoManager
		if recordUndo {
			if mgr = unison.UndoManagerFor(table); mgr != nil {
				undo = &unison.UndoEdit[*TableUndoEditData[T]]{
					ID:         unison.NextUndoID(),
					EditName:   i18n.Text("Delete Selection"),
					UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
					RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
					AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
					BeforeData: NewTableUndoEditData(table),
				}
			}
		}
		needSet := false
		topLevelData := provider.RootData()
		for _, target := range list {
			parent := gurps.AsNode(target).Parent()
			if parent == zero {
				for i, one := range topLevelData {
					if one == target {
						topLevelData = slices.Delete(topLevelData, i, i+1)
						needSet = true
						break
					}
				}
			} else {
				pNode := gurps.AsNode(parent)
				children := pNode.NodeChildren()
				for i, one := range children {
					if one == target {
						pNode.SetChildren(slices.Delete(children, i, i+1))
						break
					}
				}
			}
		}
		if needSet {
			provider.SetRootData(topLevelData)
		}
		if recordUndo && mgr != nil && undo != nil {
			undo.AfterData = NewTableUndoEditData(table)
			mgr.Add(undo)
		}
		if builder := unison.AncestorOrSelf[Rebuildable](table); builder != nil {
			builder.Rebuild(true)
		}
	}
}

// DuplicateSelection duplicates the selected nodes in the table.
func DuplicateSelection[T gurps.NodeTypes](table *unison.Table[*Node[T]]) {
	if provider, ok := any(table.Model).(TableProvider[T]); ok && !table.IsFiltered() && table.HasSelection() {
		var undo *unison.UndoEdit[*TableUndoEditData[T]]
		mgr := unison.UndoManagerFor(table)
		if mgr != nil {
			undo = &unison.UndoEdit[*TableUndoEditData[T]]{
				ID:         unison.NextUndoID(),
				EditName:   i18n.Text("Duplicate Selection"),
				UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
				RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
				AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
				BeforeData: NewTableUndoEditData(table),
			}
		}
		var zero T
		needSet := false
		topLevelData := provider.RootData()
		sel := table.SelectedRows(true)
		selMap := make(map[uuid.UUID]bool, len(sel))
		for _, row := range sel {
			if target := row.Data(); target != zero {
				tData := gurps.AsNode(target)
				parent := tData.Parent()
				clone := tData.Clone(tData.OwningEntity(), parent, false)
				selMap[gurps.AsNode(clone).UUID()] = true
				if parent == zero {
					for i, child := range topLevelData {
						if child == target {
							topLevelData = slices.Insert(topLevelData, i+1, clone)
							needSet = true
							break
						}
					}
				} else {
					pNode := gurps.AsNode(parent)
					children := pNode.NodeChildren()
					for i, child := range children {
						if child == target {
							pNode.SetChildren(slices.Insert(children, i+1, clone))
							break
						}
					}
				}
			}
		}
		if needSet {
			provider.SetRootData(topLevelData)
		}
		table.SyncToModel()
		table.SetSelectionMap(selMap)
		if mgr != nil && undo != nil {
			undo.AfterData = NewTableUndoEditData(table)
			mgr.Add(undo)
		}
		if builder := unison.AncestorOrSelf[Rebuildable](table); builder != nil {
			builder.Rebuild(true)
		}
	}
}

// CopyRowsTo copies the provided rows to the target table.
func CopyRowsTo[T gurps.NodeTypes](table *unison.Table[*Node[T]], rows []*Node[T], postProcessor func(rows []*Node[T]), recordUndo bool) {
	if table == nil || table.IsFiltered() {
		return
	}
	rows = slices.Clone(rows)
	for j, row := range rows {
		rows[j] = row.CloneForTarget(table, nil)
	}
	var undo *unison.UndoEdit[*TableUndoEditData[T]]
	var mgr *unison.UndoManager
	if recordUndo {
		if mgr = unison.UndoManagerFor(table); mgr != nil {
			undo = &unison.UndoEdit[*TableUndoEditData[T]]{
				ID:         unison.NextUndoID(),
				EditName:   fmt.Sprintf(i18n.Text("Insert %s"), gurps.AsNode(rows[0].Data()).Kind()),
				UndoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.BeforeData.Apply() },
				RedoFunc:   func(e *unison.UndoEdit[*TableUndoEditData[T]]) { e.AfterData.Apply() },
				AbsorbFunc: func(e *unison.UndoEdit[*TableUndoEditData[T]], other unison.Undoable) bool { return false },
				BeforeData: NewTableUndoEditData(table),
			}
		}
	}
	table.SetRootRows(append(slices.Clone(table.RootRows()), rows...))
	selMap := make(map[uuid.UUID]bool, len(rows))
	for _, row := range rows {
		selMap[row.UUID()] = true
	}
	table.SetSelectionMap(selMap)
	if postProcessor != nil {
		postProcessor(rows)
	}
	table.ScrollRowCellIntoView(table.LastSelectedRowIndex(), 0)
	table.ScrollRowCellIntoView(table.FirstSelectedRowIndex(), 0)
	if recordUndo && mgr != nil && undo != nil {
		undo.AfterData = NewTableUndoEditData(table)
		mgr.Add(undo)
	}
	unison.Ancestor[Rebuildable](table).Rebuild(true)
}

// DisableSorting disables the sorting capability in the table headers.
func DisableSorting[T unison.TableRowConstraint[T]](headers []unison.TableColumnHeader[T]) []unison.TableColumnHeader[T] {
	for _, header := range headers {
		state := header.SortState()
		state.Sortable = false
		header.SetSortState(state)
	}
	return headers
}

*/
