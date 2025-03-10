#!/usr/bin/env bash

set -e

# Ensure PUBLIC_GITHUB_TOKEN is set
if [ -z "${PUBLIC_GITHUB_TOKEN}" ]; then
    echo "Error: PUBLIC_GITHUB_TOKEN environment variable is not set."
    exit 1
fi

TC_PUBLIC_REPO=turbonomic-container-platform

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SRC_DIR=${SCRIPT_DIR}/../../
OUTPUT_DIR=$(mktemp -d)

if ! command -v git > /dev/null 2>&1; then
    echo "Error: git could not be found."
    exit 1
fi

echo "===> Cloning public repo...";
mkdir -p "${OUTPUT_DIR}"
cd "${OUTPUT_DIR}"
git clone https://"${PUBLIC_GITHUB_TOKEN}"@github.com/IBM/${TC_PUBLIC_REPO}.git
cd ${TC_PUBLIC_REPO}

# Cleanup install and test directories only
echo "===> Cleanup existing orm/install and orm/examples/test folders"
rm -rf orm/install
rm -rf orm/examples/test

# Recreate the required directories
mkdir -p orm/install orm/examples/test

# Copy CRD files starting with 'devops.turbonomic.io' to orm/install, excluding 'advicemappings.yaml'
echo "===> Copy CRD files"
find "${SRC_DIR}"/config/crd/bases -type f -name 'devops.turbonomic.io*' ! -name '*advicemappings.yaml' -exec cp {} orm/install/ \;

# Sync updates to orm/examples while preserving contributions
echo "===> Sync updates to ORM/examples to preserve any external contributions"
# Use rsync to add new files without overwriting existing ones to retain 3rd party contributions.
rsync -av --ignore-existing "${SRC_DIR}"/library/ orm/examples/

# Copy test files to orm/examples/test
echo "===> Copy test files to orm/examples/test"
cp -r "${SRC_DIR}"/test/* orm/examples/test/

# Commit CRDs and other files to the public repo
echo "===> Commit CRDs and other files to public repo"
git add orm/
if ! git diff --quiet --cached; then
    git commit -m "sync orm"
    git push
else
    echo "No changed files"
fi

# Cleanup
rm -rf "${OUTPUT_DIR}"

echo ""
echo "Update public repo complete."
