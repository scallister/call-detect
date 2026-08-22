//go:build linux

package tray

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/scallister/call-detect/internal/state"
)

const (
	sniPath   = "/StatusNotifierItem"
	menuPath  = "/StatusNotifierMenu"
	sniIface  = "org.kde.StatusNotifierItem"
	menuIface = "com.canonical.dbusmenu"

	dbusIDInstall   int32 = 100
	dbusIDUninstall int32 = 101
	dbusIDWebhook   int32 = 102
	dbusIDGitHub    int32 = 103
	dbusIDQuit      int32 = 104
	dbusIDUpdate    int32 = 105
)

type iconPixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

type sniTooltip struct {
	IconName string
	Pixmaps  []iconPixmap
	Title    string
	Text     string
}

type menuNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type hostImpl struct {
	mu            sync.Mutex
	snap          state.Snapshot
	webhookFailed bool
	actions       Actions
	conn          *dbus.Conn
	props         *prop.Properties
	menuProps     *prop.Properties
	menuRev       uint32
	done          chan struct{}
	once          sync.Once
}

func newHostImpl() hostImpl {
	return hostImpl{snap: SnapshotIdle(), done: make(chan struct{}), menuRev: 1}
}

func (h *hostImpl) setActions(a Actions) {
	h.mu.Lock()
	h.actions = a
	h.mu.Unlock()
}

func (h *hostImpl) update(s state.Snapshot) {
	h.mu.Lock()
	h.snap = s
	h.mu.Unlock()
	h.pushIcon()
}

func (h *hostImpl) setWebhookFailed(failed bool) {
	h.mu.Lock()
	h.webhookFailed = failed
	h.mu.Unlock()
	h.pushIcon()
}

func (h *hostImpl) run(ready func()) {
	if err := h.setup(); err != nil {
		log.Printf("tray setup failed: %v; continuing without icon", err)
		startReady(ready)
		<-h.done
		return
	}
	defer h.teardown()
	startReady(ready)
	<-h.done
}

func (h *hostImpl) quit() {
	RunExitHook()
	h.once.Do(func() {
		close(h.done)
		h.mu.Lock()
		conn := h.conn
		h.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
	})
}

func (h *hostImpl) setup() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("session bus: %w", err)
	}
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		if err == nil {
			err = fmt.Errorf("request name %s: %v", name, reply)
		}
		return err
	}

	h.mu.Lock()
	h.conn = conn
	h.mu.Unlock()

	if err := conn.Export(h, sniPath, sniIface); err != nil {
		conn.Close()
		return fmt.Errorf("export item: %w", err)
	}
	if err := conn.Export(h, menuPath, menuIface); err != nil {
		conn.Close()
		return fmt.Errorf("export menu: %w", err)
	}

	itemProps, err := prop.Export(conn, sniPath, h.itemProps())
	if err != nil {
		conn.Close()
		return fmt.Errorf("item props: %w", err)
	}
	menuProps, err := prop.Export(conn, menuPath, h.menuPropSpec())
	if err != nil {
		conn.Close()
		return fmt.Errorf("menu props: %w", err)
	}
	h.props = itemProps
	h.menuProps = menuProps

	itemNode := introspect.Node{
		Name: sniPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       sniIface,
				Methods:    introspect.Methods(h),
				Properties: itemProps.Introspection(sniIface),
				Signals: []introspect.Signal{
					{Name: "NewTitle"},
					{Name: "NewIcon"},
					{Name: "NewAttentionIcon"},
					{Name: "NewStatus", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
					{Name: "NewToolTip"},
				},
			},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(&itemNode), sniPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		conn.Close()
		return err
	}
	menuNode := introspect.Node{
		Name: menuPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       menuIface,
				Methods:    introspect.Methods(h),
				Properties: menuProps.Introspection(menuIface),
				Signals: []introspect.Signal{
					{Name: "LayoutUpdated", Args: []introspect.Arg{{Name: "revision", Type: "u"}, {Name: "parent", Type: "i"}}},
				},
			},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(&menuNode), menuPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		conn.Close()
		return err
	}

	if err := registerSNI(conn, name); err != nil {
		log.Printf("tray watcher: %v (icon appears when a StatusNotifier host is running)", err)
	}
	go h.watchWatcher(conn, name)
	return nil
}

func (h *hostImpl) teardown() {
	h.mu.Lock()
	conn := h.conn
	h.conn = nil
	h.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

func (h *hostImpl) itemProps() map[string]map[string]*prop.Prop {
	h.mu.Lock()
	pix := currentPixmap(h.webhookFailed, h.snap.Busy)
	tip := TooltipAlert(h.snap, h.webhookFailed)
	h.mu.Unlock()
	return map[string]map[string]*prop.Prop{
		sniIface: {
			"Category":      {Value: "ApplicationStatus", Writable: false, Emit: prop.EmitTrue},
			"Id":            {Value: "call-detect", Writable: false, Emit: prop.EmitTrue},
			"Title":         {Value: "call-detect", Writable: false, Emit: prop.EmitTrue},
			"Status":        {Value: "Active", Writable: false, Emit: prop.EmitTrue},
			"IconName":      {Value: "", Writable: false, Emit: prop.EmitTrue},
			"IconThemePath": {Value: "", Writable: false, Emit: prop.EmitTrue},
			"IconPixmap":    {Value: []iconPixmap{pix}, Writable: true, Emit: prop.EmitTrue},
			"ToolTip":       {Value: sniTooltip{Title: "call-detect", Text: tip}, Writable: true, Emit: prop.EmitTrue},
			"ItemIsMenu":    {Value: true, Writable: false, Emit: prop.EmitTrue},
			"Menu":          {Value: dbus.ObjectPath(menuPath), Writable: false, Emit: prop.EmitTrue},
		},
	}
}

func (h *hostImpl) menuPropSpec() map[string]map[string]*prop.Prop {
	return map[string]map[string]*prop.Prop{
		menuIface: {
			"Version":       {Value: h.menuRev, Writable: true, Emit: prop.EmitTrue},
			"TextDirection": {Value: "ltr", Writable: false, Emit: prop.EmitTrue},
			"Status":        {Value: "normal", Writable: false, Emit: prop.EmitTrue},
			"IconThemePath": {Value: []string{}, Writable: false, Emit: prop.EmitTrue},
		},
	}
}

func currentPixmap(failed, busy bool) iconPixmap {
	var data []byte
	switch {
	case failed:
		data = ErrorARGB()
	case busy:
		data = BusyARGB()
	default:
		data = IdleARGB()
	}
	return iconPixmap{Width: iconSize, Height: iconSize, Data: data}
}

func (h *hostImpl) pushIcon() {
	h.mu.Lock()
	props := h.props
	conn := h.conn
	pix := currentPixmap(h.webhookFailed, h.snap.Busy)
	tip := TooltipAlert(h.snap, h.webhookFailed)
	h.menuRev++
	rev := h.menuRev
	menuProps := h.menuProps
	h.mu.Unlock()
	if props == nil {
		return
	}
	_ = props.Set(sniIface, "IconPixmap", dbus.MakeVariant([]iconPixmap{pix}))
	_ = props.Set(sniIface, "ToolTip", dbus.MakeVariant(sniTooltip{Title: "call-detect", Text: tip}))
	if menuProps != nil {
		_ = menuProps.Set(menuIface, "Version", dbus.MakeVariant(rev))
	}
	if conn != nil {
		_ = conn.Emit(sniPath, sniIface+".NewIcon")
		_ = conn.Emit(sniPath, sniIface+".NewToolTip")
		_ = conn.Emit(menuPath, menuIface+".LayoutUpdated", rev, int32(0))
	}
}

func registerSNI(conn *dbus.Conn, service string) error {
	var last error
	for _, dest := range []string{"org.kde.StatusNotifierWatcher", "org.freedesktop.StatusNotifierWatcher"} {
		obj := conn.Object(dest, "/StatusNotifierWatcher")
		if err := obj.Call(dest+".RegisterStatusNotifierItem", 0, service).Err; err == nil {
			return nil
		} else {
			last = err
		}
		if err := obj.Call(dest+".RegisterStatusNotifierItem", 0, sniPath).Err; err == nil {
			return nil
		} else {
			last = err
		}
	}
	if last == nil {
		return fmt.Errorf("no StatusNotifierWatcher")
	}
	return last
}

func (h *hostImpl) watchWatcher(conn *dbus.Conn, service string) {
	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
	); err != nil {
		return
	}
	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	for sig := range ch {
		if sig == nil || sig.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(sig.Body) < 3 {
			continue
		}
		name, _ := sig.Body[0].(string)
		newOwner, _ := sig.Body[2].(string)
		if newOwner == "" {
			continue
		}
		if name == "org.kde.StatusNotifierWatcher" || name == "org.freedesktop.StatusNotifierWatcher" {
			if err := registerSNI(conn, service); err != nil {
				log.Printf("tray watcher: %v", err)
			}
		}
	}
}

func (h *hostImpl) ContextMenu(x, y int32) *dbus.Error {
	go h.dialogMenu()
	return nil
}

func (h *hostImpl) Activate(x, y int32) *dbus.Error {
	go h.dialogMenu()
	return nil
}

func (h *hostImpl) SecondaryActivate(x, y int32) *dbus.Error { return nil }

func (h *hostImpl) Scroll(delta int32, orientation string) *dbus.Error { return nil }

func (h *hostImpl) dialogMenu() {
	h.mu.Lock()
	snap := h.snap
	failed := h.webhookFailed
	actions := h.actions
	h.mu.Unlock()
	id, ok := pickMenu(statusLines(snap, failed), actionChoices(actions))
	if ok {
		handleMenuID(actions, id, h.quit)
	}
}

func (h *hostImpl) GetLayout(parentID int32, recursionDepth int32, propertyNames []string) (uint32, menuNode, *dbus.Error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if parentID != 0 {
		return h.menuRev, menuNode{ID: parentID, Properties: map[string]dbus.Variant{}, Children: []dbus.Variant{}}, nil
	}
	return h.menuRev, h.buildMenuLocked(), nil
}

func (h *hostImpl) GetGroupProperties(ids []int32, propertyNames []string) (out []struct {
	V0 int32
	V1 map[string]dbus.Variant
}, err *dbus.Error) {
	root := h.buildMenuCopy()
	byID := map[int32]map[string]dbus.Variant{root.ID: root.Properties}
	for _, child := range root.Children {
		if n, ok := child.Value().(menuNode); ok {
			byID[n.ID] = n.Properties
		}
	}
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, struct {
				V0 int32
				V1 map[string]dbus.Variant
			}{id, p})
		}
	}
	return out, nil
}

func (h *hostImpl) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	root := h.buildMenuCopy()
	if id == 0 {
		return root.Properties[name], nil
	}
	for _, child := range root.Children {
		if n, ok := child.Value().(menuNode); ok && n.ID == id {
			return n.Properties[name], nil
		}
	}
	return dbus.MakeVariant(""), nil
}

func (h *hostImpl) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID == "clicked" {
		go h.handleDbusID(id)
	}
	return nil
}

func (h *hostImpl) EventGroup(events []struct {
	V0 int32
	V1 string
	V2 dbus.Variant
	V3 uint32
}) ([]int32, *dbus.Error) {
	for _, ev := range events {
		if ev.V1 == "clicked" {
			go h.handleDbusID(ev.V0)
		}
	}
	return nil, nil
}

func (h *hostImpl) AboutToShow(id int32) (bool, *dbus.Error) { return true, nil }

func (h *hostImpl) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return nil, nil, nil
}

func (h *hostImpl) handleDbusID(id int32) {
	h.mu.Lock()
	actions := h.actions
	h.mu.Unlock()
	switch id {
	case dbusIDInstall:
		handleMenuID(actions, menuInstall, h.quit)
	case dbusIDUninstall:
		handleMenuID(actions, menuUninstall, h.quit)
	case dbusIDWebhook:
		handleMenuID(actions, menuWebhook, h.quit)
	case dbusIDUpdate:
		handleMenuID(actions, menuUpdate, h.quit)
	case dbusIDGitHub:
		handleMenuID(actions, menuGitHub, h.quit)
	case dbusIDQuit:
		handleMenuID(actions, menuQuit, h.quit)
	}
}

func (h *hostImpl) buildMenuCopy() menuNode {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.buildMenuLocked()
}

func (h *hostImpl) buildMenuLocked() menuNode {
	var children []dbus.Variant
	id := int32(1)
	for _, line := range statusLines(h.snap, h.webhookFailed) {
		children = append(children, dbus.MakeVariant(disabledNode(id, line)))
		id++
	}
	children = append(children, dbus.MakeVariant(separatorNode(id)))
	id++
	for _, c := range actionChoices(h.actions) {
		children = append(children, dbus.MakeVariant(actionNode(dbusIDFor(c.ID), c.Label)))
	}
	return menuNode{
		ID:         0,
		Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")},
		Children:   children,
	}
}

func dbusIDFor(id string) int32 {
	switch id {
	case menuInstall:
		return dbusIDInstall
	case menuUninstall:
		return dbusIDUninstall
	case menuWebhook:
		return dbusIDWebhook
	case menuUpdate:
		return dbusIDUpdate
	case menuGitHub:
		return dbusIDGitHub
	case menuQuit:
		return dbusIDQuit
	default:
		return 0
	}
}

func disabledNode(id int32, label string) menuNode {
	return menuNode{
		ID: id,
		Properties: map[string]dbus.Variant{
			"label":   dbus.MakeVariant(escapeMenuLabel(label)),
			"enabled": dbus.MakeVariant(false),
		},
	}
}

func actionNode(id int32, label string) menuNode {
	return menuNode{
		ID: id,
		Properties: map[string]dbus.Variant{
			"label":   dbus.MakeVariant(escapeMenuLabel(label)),
			"enabled": dbus.MakeVariant(true),
		},
	}
}

func separatorNode(id int32) menuNode {
	return menuNode{
		ID:         id,
		Properties: map[string]dbus.Variant{"type": dbus.MakeVariant("separator")},
	}
}

func escapeMenuLabel(s string) string {
	return strings.ReplaceAll(s, "_", "__")
}
