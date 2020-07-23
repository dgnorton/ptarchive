#!/bin/sh

set -e

ver="$(git tag --points-at HEAD)"

if [ -z "${ver}" ]; then
	ver="$(git rev-parse --short HEAD)"
fi

if [ -z "${ver}" ]; then
	echo "error: no git tag or sha found"
	exit 1
fi

outDir="${PWD}"

# Create a temp staging directory.
stagingDir="$(mktemp -d 2>/dev/null || mktemp -d -t 'ptarchive_')"
echo "staging directory: ${stagingDir}"

# Build the binaries.
echo "building binaries..."
GOOS=linux go build -o ${stagingDir}/ptarchive_linux
GOOS=darwin go build -o ${stagingDir}/ptarchive_mac
GOOS=windows go build -o ${stagingDir}/ptarchive_windows

# Package the binaries.
echo "zipping binaries..."
zip -j ptarchive_bin_${ver}.zip  ${stagingDir}/ptarchive_linux ${stagingDir}/ptarchive_mac ${stagingDir}/ptarchive_windows

# Clone the source code repo.
echo "cloning source code..."
git clone --depth=1 --branch master https://github.com/dgnorton/ptarchive ${stagingDir}/ptarchive
rm -rf ${stagingDir}/ptarchive/.git

# Zip the source code.
echo "zipping source code..."
cd ${stagingDir} && zip -r ${outDir}/ptarchive_src_${ver}.zip ptarchive/

# Delete the staging directory.
rm -rf ${stagingDir}

echo "done"
