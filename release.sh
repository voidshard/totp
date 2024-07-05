#! /bin/sh

set -ex

VERSION=0.0.1

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR

docker build -t totp:${VERSION} -f Dockerfile .
docker tag totp:${VERSION} uristmcdwarf/totp:${VERSION}
docker push uristmcdwarf/totp:${VERSION}

cd -
