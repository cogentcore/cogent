# Mac install

app-install:
	/bin/cp -av Gide.app /Applications
	- mkdir /Applications/Gide.app/Contents/MacOS/
	/bin/cp $(HOME)/go/bin/gide /Applications/Gide.app/Contents/MacOS/

dev-install:
	/bin/cp -av Gide-dev.app /Applications/Gide.app
	- mkdir /Applications/Gide.app/Contents/MacOS/
	/bin/ln -s $(HOME)/go/bin/gide /Applications/Gide.app/Contents/MacOS/

app-dmg:
	/bin/cp $(HOME)/go/bin/gide Gide.app/Contents/MacOS/
	/usr/bin/hdiutil create -ov -srcfolder Gide.app Gide.dmg

