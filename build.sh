#!/bin/bash

# do local push based on inputs
INPUT=$1
PUSH=${INPUT:-"1"}

# check requirements
EXIT_NOW=0
which buildah > /dev/null 2>&1
if [ $? -eq 1 ]; then
	printf "The 'buildah' executable is required for this build tool.\n"
	printf "  Fedora: 'sudo dnf install -y buildah'\n"
	printf "  See: 'https://github.com/projectatomic/buildah/blob/master/install.md'\n"
	EXIT_NOW=1
fi
which runc > /dev/null 2>&1
if [ $? -eq 1 ]; then
	printf "The 'runc' executable is required for this build tool.\n"
	printf "  Fedora: 'sudo dnf install -y runc'\n"
	printf "  See: 'https://github.com/opencontainers/runc'\n"
	EXIT_NOW=1
fi
which podman > /dev/null 2>&1
if [ $? -eq 1 ]; then
	printf "The 'podman' executable is required for this build tool.\n"
	printf "  Fedora: 'sudo dnf install -y podman'\n"
	printf "  See: 'https://github.com/containers/libpod'\n"
	EXIT_NOW=1
fi

# exit if we need to
if [ $EXIT_NOW -eq 1 ]; then
	exit $EXIT_NOW
fi

# get version from version file
VERSION=$(cat ./.version)
GITHASH=$(git rev-parse HEAD | head -c6)
# export output
export MAJOR_TAG="${VERSION}"
export BUILD_TAG="${VERSION}-git${GITHASH}"

TARGET=$(pwd)/target
# remove and remake output target
rm -rf $TARGET
mkdir $TARGET

# build a golang container to build the binary
GOVERSION="1.10"
GOLANG_CONTAINER_ROOT="/go/src/github.com/chrisruffalo/gyip"
GOLANG_CONTAINER=$(buildah from golang:${GOVERSION}-alpine)
buildah umount $GOLANG_CONTAINER # ensure unmounted
buildah run $GOLANG_CONTAINER -- mkdir -p $GOLANG_CONTAINER_ROOT{,/command}
buildah config --workingdir "${GOLANG_CONTAINER_ROOT}" --env CGO_ENABLED="0" $GOLANG_CONTAINER
buildah copy $GOLANG_CONTAINER .version $GOLANG_CONTAINER_ROOT
buildah copy $GOLANG_CONTAINER *.go $GOLANG_CONTAINER_ROOT
buildah copy $GOLANG_CONTAINER command/ $GOLANG_CONTAINER_ROOT/command
buildah run $GOLANG_CONTAINER -- apk add --no-cache git > /dev/null 2>&1
buildah run $GOLANG_CONTAINER -- go get
buildah run $GOLANG_CONTAINER -- go build -a -tags netgo -ldflags "-w -X main.Version=${VERSION} -X main.GitHash=${GITHASH} -extldflags \"-static\"" -o gyip
MOUNT_DIR=$(buildah mount $GOLANG_CONTAINER)
cp "${MOUNT_DIR}${GOLANG_CONTAINER_ROOT}/gyip" "$TARGET/gyip"
buildah umount $GOLANG_CONTAINER # remove created mount
buildah rm $GOLANG_CONTAINER # clean up

# build the actual container by shoving the binary into a minimal container (scratch or alpine)
BASE_CONTAINER=${BASE_CONTAINER:-"scratch"}
GYIP_CONTAINER=$(buildah from ${BASE_CONTAINER})
GYIP_CONTAINER_PATH="gyip/gyip"
GYIP_CONTAINER_TAG="${GYIP_CONTAINER_PATH}:${BUILD_TAG}"
buildah config --port 8053/tcp --port 8053/udp --workingdir "/" --entrypoint '["/gyip","--host 0.0.0.0"]' $GYIP_CONTAINER
buildah copy $GYIP_CONTAINER "$TARGET/gyip" /gyip
buildah commit $GYIP_CONTAINER $GYIP_CONTAINER_TAG
# more tags
buildah tag $GYIP_CONTAINER_TAG $GYIP_CONTAINER_PATH:$MAJOR_TAG
# remove working container
buildah rm $GYIP_CONTAINER 

# local push if true (default)
if [ $PUSH -eq 1 ]; then
	# push target output containers
	buildah push $GYIP_CONTAINER_TAG oci:$TARGET/oci-gyip:${BUILD_TAG}
	buildah push $GYIP_CONTAINER_TAG oci-archive:$TARGET/oci-gyip.tar:${BUILD_TAG}
	buildah push $GYIP_CONTAINER_TAG docker-archive:$TARGET/docker-gyip.tar:$GYIP_CONTAINER_TAG
fi