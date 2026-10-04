Require pomo

Set Padding 20
Set Margin 20
Set BorderRadius 10

Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set MarginFill "#0D1117"

Set FontSize 18
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 1920
Set Height 1200

# 1. mono12
Hide
Type "sed -i 's/font:.*/font: mono12/' pomo.yaml"
Enter
Show
Type "pomo 25m"
Enter
Sleep 1.5s
Screenshot .github/assets/mono12.png
Type "q"
Sleep 1s

# 2. rebel
Hide
Type "sed -i 's/font:.*/font: rebel/' pomo.yaml"
Enter
Show
Type "pomo 25m"
Enter
Sleep 1.5s
Screenshot .github/assets/rebel.png
Type "q"
Sleep 1s

# 3. ansi
Hide
Type "sed -i 's/font:.*/font: ansi/' pomo.yaml"
Enter
Show
Type "pomo 25m"
Enter
Sleep 1.5s
Screenshot .github/assets/ansi.png
Type "q"
Sleep 1s

# 4. ansiShadow
Hide
Type "sed -i 's/font:.*/font: ansiShadow/' pomo.yaml"
Enter
Show
Type "pomo 25m"
Enter
Sleep 1.5s
Screenshot .github/assets/ansiShadow.png
Type "q"
Sleep 1s

# Restore default font
Hide
Type "sed -i 's/font:.*/font: mono12/' pomo.yaml"
Enter
