Output build/demo.gif

Set Shell bash
Set FontSize 15
Set Width 1240
Set Height 600
Set Padding 24
Set TypingSpeed 35ms
Set Theme "GitHub Dark"

Hide
Type "cd $DEMO_DIR && export PATH=$DEMO_BIN:$DEMO_DIR/.venv/bin:$PATH PS1='$ ' && clear"
Enter
Show

Type "cat spec.md"
Enter
Sleep 3s

Type "pytest -q --junitxml=junit.xml"
Enter
Sleep 2s

Type "shallnot check --specs spec.md --tests tests --results junit.xml"
Enter
Sleep 6s

Type "clear && cat username-rule.patch | grep '^+' | tail -9"
Enter
Sleep 5s

Type "patch -p1 -s < username-rule.patch && pytest -q --junitxml=junit.xml"
Enter
Sleep 2s

Type "clear && shallnot check --specs spec.md --tests tests --results junit.xml"
Enter
Sleep 7s
