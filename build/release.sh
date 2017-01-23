#!/bin/bash

VERSION="0.9.0"

BUILD_PATH="$PWD"
#BUILD_PATH="/tmp/dmoni/build"

REPO_PATH="github.com/lizhongz/dmoni"
FOLDER=dmoni-$VERSION
BIN=$FOLDER/dmoni
SNAPSHOT=$FOLDER/snapshot
README=$FOLDER/README.md

# Check if build dir exists
if [ ! -d "$BUILD_PATH" ]; then
    mkdir -p $BUILD_PATH
fi

cd $BUILD_PATH
mkdir $FOLDER

# Build DMoni
go build -o $BIN $REPO_PATH 

# Copy snapshot repo
git clone --depth=1 git@gitlab.ncsl.nist.gov:lnz5/snapshot.git $SNAPSHOT >/dev/null && rm -rf $SNAPSHOT/.git

# Copy README
cp $GOPATH/src/$REPO_PATH/README.md $README

# Create a tarball
tar -zcvf $FOLDER.tar.gz $FOLDER >/dev/null
 
# Clean up
rm -rf $FOLDER

cd - >/dev/null
