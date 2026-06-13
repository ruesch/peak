# Peak Commands Reference

Commands can be typed in any tag or body and executed by middle-clicking.


## 1. Global Commands

- NewCol             Create a new column.
- Help               Open the documentation index.
- Exit               Quit Peak. Warns if any window has unsaved changes.


## 2. Column Commands

- New [path]         Open a file in this column, or create a new empty window.
- Win [cmd]          Open a terminal window in this column.
- Zerox              Duplicate the focused window into this column.
- Delcol             Close the column. Warns if any window has unsaved changes.
- Sort               Sort windows in the column alphabetically by filename.


## 3. Window Commands

- Get [path]         Reload the current file from disk, or load path.
- Put [path]         Save the buffer to disk, or save as path.
- Undo / Redo        Undo or redo the last text modification.
- Snarf              Copy the selection to the clipboard.
- Cut                Cut the selection to the clipboard.
- Paste              Paste from the clipboard at the selection or cursor.
- Look [pattern]     Search for pattern in the body. Uses the selection
                     if no pattern is given.
- Edit <expr>        Apply a structural regex command. See StructuralRegex.md.
- Tab [n]            Set the tab width to n, or show the current tab width.
- Theme [name]       Apply a color theme, or open the theme list.
- Zerox              Duplicate this window.
- Del                Close the window. Warns if there are unsaved changes.
- Delete             Force-close the window without checking for unsaved
                     changes.


## 4. Filesystem Commands

- Mount socket path  Mount a 9P socket at path.
- Bind src dest      Bind src into dest in the virtual filesystem.
- Umount path        Unmount path.


## 5. External Commands

Any word or selection that is not a built-in is run as a shell command.
Output appears in a +Errors window.

Pipe operators work on the current selection:

- <cmd               Replace the selection with the command output.
- >cmd               Pipe the selection into the command (discards output).
- |cmd               Pipe the selection through the command and replace it.


## 6. Path Syntax

Relative paths are resolved from the current window's directory.

- ~                  Expands to the home directory.
- +<name>            Output window (holds command output). +Errors is
                     the default for shell and command output.
- -<name>            Terminal window.


## 7. Arguments

Commands take arguments in this order:

1. Text following the command name (e.g., Get main.go).
2. Current text selection.
3. The filename in the window tag (for Get and Put).
