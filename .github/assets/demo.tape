Output .github/assets/demo.gif

Require pomo

Set Padding 20
Set Margin 20
Set BorderRadius 10

Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set MarginFill "#0D1117"

Set FontSize 18
Set FontFamily "JetBrainsMono Nerd Font"
Set TypingSpeed 60ms

Set Framerate 60
Set Width 1200
Set Height 800

Sleep 1s

# 1. Quick Task Creation via CLI
Type "pomo task add 'Read Chapter 4' -d 'Operating systems & memory management'"
Sleep 500ms
Enter
Sleep 1.5s

# 2. Launch pomo attached to task #1
Type "pomo --task 1 5m"
Sleep 500ms
Enter
Sleep 3s

# 3. Pause & Adjust time
Space
Sleep 1s
Up
Sleep 200ms
Up
Sleep 200ms
Space
Sleep 2s

# 4. Open In-TUI Task Picker (t)
Type "t"
Sleep 2s

# 5. Create new task via In-TUI Modal (n)
Type "n"
Sleep 1s
Type "Review study flashcards"
Sleep 800ms
Enter
Sleep 3s

# 6. Complete active task (c)
Type "c"
Sleep 2s

# 7. Quit and show Session Summary (q)
Type "q"
Sleep 5s
