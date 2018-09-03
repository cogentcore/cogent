#!/bin/sh
# install this script in /usr/local/bin to have command-line access to the .app

absPath() {
    if [[ -d "$1" ]]; then
        cd "$1"
        echo "$(pwd -P)"
    else 
        cd "$(dirname "$1")"
        echo "$(pwd -P)/$(basename "$1")"
    fi
}

if [ $# -eq 0 ]; then
    /Applications/gide.app/Contents/MacOS/gide &
else    
    nwargs=""
    for ar in "$@"
    do
	ap=$(absPath "$ar")
#	echo "$ar = $ap"
	if [[ "$nwargs" == "" ]]; then
	    nwargs="$ap"
	else
	    nwargs="$nwargs $ap"
	fi
    done
    /Applications/gide.app/Contents/MacOS/gide $nwargs &
fi
# wait for jobs to finish so it looks like a normal process
wait



