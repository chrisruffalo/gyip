#!/bin/bash

# check requirements
which buildah > /dev/null 2>&1
if [ $? -eq 1 ]; then
	printf "The 'buildah' executable is required for this build tool.\n"
	printf "Fedora: 'sudo dnf install -y buildah'"
	printf "See: 'https://github.com/projectatomic/buildah/blob/master/install.md"
	exit 1
fi

TARGET=$(pwd)/target
# remove and remake output target
rm -rf $TARGET
mkdir $TARGET

# build a golang container to build the binary
GOLANG_CONTAINER_ROOT="/go/src/github.com/chrisruffalo/gyip"
GOLANG_CONTAINER=$(buildah from golang:1.9-alpine)
buildah umount $GOLANG_CONTAINER # ensure unmounted
buildah run $GOLANG_CONTAINER -- mkdir -p $GOLANG_CONTAINER_ROOT{,/command}
buildah config --workingdir "${GOLANG_CONTAINER_ROOT}" --env CGO_ENABLED="0" $GOLANG_CONTAINER
buildah copy $GOLANG_CONTAINER *.go $GOLANG_CONTAINER_ROOT
buildah copy $GOLANG_CONTAINER command/ $GOLANG_CONTAINER_ROOT/command
buildah run $GOLANG_CONTAINER -- apk add --no-cache git > /dev/null 2>&1
buildah run $GOLANG_CONTAINER -- go get
buildah run $GOLANG_CONTAINER -- go build -a -tags netgo -ldflags '-w -extldflags \"-static\"' -o gyip
MOUNT_DIR=$(buildah mount $GOLANG_CONTAINER)
cp "${MOUNT_DIR}${GOLANG_CONTAINER_ROOT}/gyip" "$TARGET/gyip"
buildah umount $GOLANG_CONTAINER # remove created mount
buildah rm $GOLANG_CONTAINER # clean up

# build the actual container by shoving the binary into a minimal alpine container
GYIP_CONTAINER=$(buildah from scratch)
GYIP_CONTAINER_TAG="gyip/gyip"
buildah copy $GYIP_CONTAINER "$TARGET/gyip" /gyip
buildah config --port 8053 --workingdir "/" --entrypoint '["/gyip"]' $GYIP_CONTAINER
buildah commit $GYIP_CONTAINER $GYIP_CONTAINER_TAG
buildah rm $GYIP_CONTAINER # remove working container

# push target output containers
buildah push $GYIP_CONTAINER_TAG:latest oci:$TARGET/oci-gyip:latest
buildah push $GYIP_CONTAINER_TAG:latest oci-archive:$TARGET/oci-gyip.tar:latest
buildah push $GYIP_CONTAINER_TAG:latest docker-archive:$TARGET/docker-gyip.tar:$GYIP_CONTAINER_TAG:latest