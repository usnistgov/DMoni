#!/bin/bash

VERSION="0.9.0"

TMP_DIR="$PWD/dmoni-$VERSION"
REPO="github.com/usnistgov/DMoni"
SRC_DIR="$GOPATH/src/$REPO"

# Check if build dir exists
if [ ! -d "$TMP_DIR" ]; then
    mkdir -p $TMP_DIR/bin
fi

# Build DMoni
go build -o $TMP_DIR/bin/dmoni $REPO

# Copy snapshot repo
git clone --depth=1 git@gitlab.ncsl.nist.gov:lnz5/snapshot.git \
    $TMP_DIR/snapshot >/dev/null && rm -rf $TMP_DIR/snapshot/.git

cp $SRC_DIR/README.md $TMP_DIR
cp $SRC_DIR/tools/dmoni_manager.sh $SRC_DIR/tools/dmoni_agent.sh $TMP_DIR/bin

# Create a tarball
tar -zcvf dmoni-$VERSION.tar.gz -C $TMP_DIR/.. $(basename $TMP_DIR) >/dev/null
 
# Clean up
rm -rf $TMP_DIR
