Output media/demo.gif

Set FontSize 15
Set Width 1000
Set Height 600
Set Framerate 24
Set CursorBlink false

Type "Exporting entries from .org files to a CSV:
"
Ctrl+c
Sleep 2s

Type "worklog sync work-notes.org --sink time-entries.csv --start-date '2023-01-01'"
Sleep 500ms
Enter
Sleep 2s
Type "y"
Enter
Sleep 5s
Type "clear"
Enter

Type "Running sync commands again never creates duplicates:
"
Ctrl+c
Sleep 2s

Type "worklog sync work-notes.org --sink time-entries.csv --start-date '2023-01-01'"
Sleep 500ms
Enter
Sleep 5s
Type "clear"
Enter

Type "Consult the tool's inbuilt documentation for additional commands:
"
Ctrl+c
Sleep 2s

Type "worklog -h"
Sleep 500ms
Enter

Sleep 5s