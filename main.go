package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// version identifies the running build. Release builds overwrite it with the git
// tag via -ldflags "-X main.version=..." (see the Makefile); dev builds keep "dev".
var version = "dev"

// menuAction describes one shortcut exposed in the menu bar. The identifier is the
// contract with the frontend: runShortcut in frontend/src/lib/shortcuts.ts performs
// the matching action, and shortcutForEvent there maps the same accelerator. An empty
// label marks a separator.
type menuAction struct {
	identifier  string
	label       string
	accelerator *keys.Accelerator
}

// separator is a blank entry that renders as a divider between groups.
var separator = menuAction{}

// addActions appends each action to the submenu, wiring its callback to announce the
// identifier on the "menu:action" event. Every accelerator carries a modifier: macOS
// routes a bare-letter accelerator to the menu bar before the webview, which would
// swallow that letter whenever the user is typing.
func addActions(submenu *menu.Menu, app *App, actions []menuAction) {
	for _, action := range actions {
		if action.label == "" {
			submenu.AddSeparator()
			continue
		}
		identifier := action.identifier
		submenu.AddText(action.label, action.accelerator, func(_ *menu.CallbackData) {
			runtime.EventsEmit(app.ctx, "menu:action", identifier)
		})
	}
}

// buildAppMenu assembles the native macOS menu bar, listing every shortcut the app
// offers so they are discoverable. The standard App menu supplies Quit (cmd+q); the
// Edit menu restores the system clipboard shortcuts inside the webview; the Window
// menu adds minimise/zoom. This runs before startup, so app.ctx is still nil here —
// the callbacks only read it when fired, which is always after startup.
func buildAppMenu(app *App) *menu.Menu {
	applicationMenu := menu.NewMenu()
	applicationMenu.Append(menu.AppMenu())

	// Settings leads the File menu because Wails' stock App menu is a fixed role-based
	// item we cannot insert the conventional macOS Settings entry into.
	fileMenu := applicationMenu.AddSubmenu("File")
	addActions(fileMenu, app, []menuAction{
		{"app.settings", "Settings…", keys.CmdOrCtrl(",")},
		separator,
		{"project.new", "New Project", keys.CmdOrCtrl("n")},
		{"project.edit", "Edit Project", keys.Combo("e", keys.CmdOrCtrlKey, keys.ShiftKey)},
		{"project.delete", "Delete Project", keys.Combo("backspace", keys.CmdOrCtrlKey, keys.ShiftKey)},
		separator,
		{"editor.save", "Save", keys.CmdOrCtrl("s")},
	})
	fileMenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		runtime.Quit(app.ctx)
	})

	// Wails' stock Edit menu supplies the clipboard roles, and its own Undo/Redo
	// roles already carry cmd+z and shift+cmd+z. Graph undo is therefore listed
	// without accelerators: registering the same combination twice would make which
	// item wins ambiguous, and the webview keydown handler in App.svelte is what
	// actually drives it.
	applicationMenu.Append(menu.EditMenu())

	historyMenu := applicationMenu.AddSubmenu("History")
	addActions(historyMenu, app, []menuAction{
		{"edit.undo", "Undo        ⌘Z", nil},
		{"edit.redo", "Redo        ⇧⌘Z", nil},
	})

	nodeMenu := applicationMenu.AddSubmenu("Node")
	addActions(nodeMenu, app, []menuAction{
		{"node.precursor", "Add Precursor", keys.CmdOrCtrl("t")},
		{"node.newTask", "New Endpoint Task", keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey)},
		{"node.decision", "Add Decision", keys.CmdOrCtrl("d")},
		separator,
		{"node.edit", "Edit Node", keys.CmdOrCtrl("e")},
		{"details.show", "Details", keys.CmdOrCtrl("i")},
		separator,
		{"node.group", "Group Selection", keys.CmdOrCtrl("g")},
		{"node.delete", "Delete Selection", keys.CmdOrCtrl("backspace")},
	})

	viewMenu := applicationMenu.AddSubmenu("View")
	addActions(viewMenu, app, []menuAction{
		{"view.home", "Home", keys.CmdOrCtrl("0")},
		{"view.fit", "Fit All", keys.Combo("0", keys.CmdOrCtrlKey, keys.ShiftKey)},
		{"view.chainPrev", "Previous Chain", keys.CmdOrCtrl("8")},
		{"view.chainNext", "Next Chain", keys.CmdOrCtrl("9")},
		{"view.find", "Find on Canvas", keys.CmdOrCtrl("f")},
	})
	// Escape is shown for discoverability but never registered: as a bare accelerator
	// the menu bar would intercept it before the webview's dismiss cascade ran.
	escapeItem := viewMenu.AddText("Close / Cancel        Esc", nil, nil)
	escapeItem.Disabled = true

	applicationMenu.Append(menu.WindowMenu())
	return applicationMenu
}

func main() {
	// Create an instance of the app structure
	app := NewApp(version)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Precursor",
		Width:     1280,
		Height:    860,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 41, B: 59, A: 1},
		Menu:             buildAppMenu(app),
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
