# Mac install

app-install:
	/bin/cp -av Grid.app /Applications
	- mkdir /Applications/Grid.app/Contents/MacOS/
	/bin/cp $(HOME)/go/bin/grid /Applications/Grid.app/Contents/MacOS/

dev-install:
	/bin/cp -av Grid-dev.app /Applications/Grid.app
	- mkdir /Applications/Grid.app/Contents/MacOS/
	/bin/ln -s $(HOME)/go/bin/grid /Applications/Grid.app/Contents/MacOS/

app-dmg:
	/bin/cp $(HOME)/go/bin/grid Grid.app/Contents/MacOS/
	/usr/bin/hdiutil create -ov -srcfolder Grid.app Grid.dmg

