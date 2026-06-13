# Structural Regular Expressions

Peak's Edit command implements a structural regular expression engine,
similar to the one in Acme and Sam. Unlike line-oriented tools like sed,
it operates on arbitrary ranges of text and composes commands recursively.


## 1. Using Edit

Type a command in any tag or body and middle-click it:

    Edit <command>

Edit operates on the current selection, or the whole file if nothing is
selected. The , address (whole file) is commonly used explicitly:

    Edit , <command>


## 2. Addresses

Addresses specify which part of the text a command operates on.

- .                 The current selection.
- 0                 The beginning of the file.
- $                 The end of the file.
- #n                Character offset n.
- n                 Line n.
- /re/              The next match of regular expression re.
- ?re?              The previous match of re.
- +                 One line forward from the current address.
- -                 One line backward from the current address.
- a1,a2             From the start of a1 to the end of a2.
- a1;a2             Like a1,a2, but evaluates a2 with a1 as context.

An empty pattern // reuses the last regular expression.


## 3. Basic Commands

- a/text/           Append text after the addressed range.
- i/text/           Insert text before the addressed range.
- c/text/           Replace the addressed range with text.
- d                 Delete the addressed range.
- p                 Print the addressed range to +Errors.
- s/re/text/        Substitute the first match of re with text.
- s/re/text/g       Substitute all matches of re with text.
- sn/re/text/       Substitute the nth match of re (n >= 1).
- w [file]          Write the whole file to file, or to the current file.
- e [file]          Replace the entire buffer with the contents of file.
- r [file]          Replace the addressed range with the contents of file.
- m addr            Move the addressed range to after addr.
- t addr            Copy the addressed range to after addr.
- u [n]             Undo the last n modifications (default 1).
                    Use u- or u-n to redo.
- f [file]          Set the window filename, or print it if no argument.
- =                 Print the line number of the addressed range.
- = #               Print the character offset of the addressed range.
- = +               Print the line+character offset of the addressed range.

In substitution text, & expands to the whole match, \1 through \9 to
the corresponding subgroups, and \n to a newline.

For a, i, and c, text can also be given as a multi-line block: omit the
delimiter and start the text on the next line, terminated by a line
containing only a dot:

    a
    first line
    second line
    .


## 4. Structural Commands

- x/re/ cmd         For each match of re in the range, run cmd.
                    Defaults to p if no command is given.
- x cmd             For each line in the range, run cmd.
- y/re/ cmd         Run cmd on each region between matches of re.
                    Defaults to p if no command is given.
- y cmd             Run cmd on each line (same as x without a pattern).
- g/re/ cmd         If the range matches re, run cmd. Defaults to p.
- v/re/ cmd         If the range does not match re, run cmd. Defaults to p.
- { ... }           Group multiple commands on the same address.


## 5. Multi-file Commands

- X/re/ cmd         Run cmd on each window whose filename matches re.
                    Defaults to f if no command is given.
- X cmd             Run cmd on every open window.
- Y/re/ cmd         Run cmd on each window whose filename does not
                    match re. Defaults to f if no command is given.
- B file ...        Open files.
- D [file ...]      Close files, or the current window if no argument.
- b [win]           Switch context to window win, or print current filename.


## 6. Shell Commands

- |cmd              Pipe the range through cmd and replace it with the output.
- >cmd              Pipe the range to cmd (discard output).
- <cmd              Replace the range with the output of cmd.
- !cmd              Run cmd in the shell (output goes to +Errors).


## 7. Examples

Delete all trailing whitespace:

    Edit , x/[ \t]+\n/ s/[ \t]+\n/\n/

Comment out every line containing "TODO":

    Edit , x/.*TODO.*\n/ i/\/\/ /

Rename a field across all Go files:

    Edit X/\.go$/ , x/OldName/ c/NewName/

Print the line number of every match of "error":

    Edit , x/error/ =

Uppercase all occurrences of "peak":

    Edit , x/peak/ |tr a-z A-Z

Wrap each match of a pattern in parentheses using a back-reference:

    Edit , s/foo(\w+)/(\1)/g
