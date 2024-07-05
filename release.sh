#! /bin/sh

set -ex

VERSION=0.0.3

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR

# build, tag the image
docker build -t totp:${VERSION} -f Dockerfile .
docker tag totp:${VERSION} uristmcdwarf/totp:${VERSION}

# set latest tag
docker tag totp:${VERSION} uristmcdwarf/totp:latest

# push the image
docker push uristmcdwarf/totp:${VERSION}
docker push uristmcdwarf/totp:latest

cd -
