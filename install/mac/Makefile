# Mac install

app-install:
	/bin/cp $(HOME)/go/bin/gide Gide.app/Contents/MacOS/
	/bin/cp -av Gide.app /Applications

dev-install:
	/bin/cp -av Gide-dev.app /Applications/Gide.app

app-dmg:
	/bin/cp $(HOME)/go/bin/gide Gide.app/Contents/MacOS/
	/usr/bin/hdiutil create -ov -srcfolder Gide.app Gide.dmg

