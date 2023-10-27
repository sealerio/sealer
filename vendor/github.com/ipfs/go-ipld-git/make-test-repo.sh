#!/usr/bin/env bash

set -x
CUR_DIR=$(pwd)
TEST_DIR=$(mktemp -d)
cd ${TEST_DIR}

git init

# Test generic commit/blob

git config user.name "John Doe"
git config user.email johndoe@example.com

echo "Hello world" > file
git add file

git commit -m "Init"

# Test generic commit/tree/blob, weird person info

mkdir dir
mkdir dir/subdir
mkdir dir2

echo "qwerty" > dir/f1
echo "123456" > dir/subdir/f2
echo "',.pyf" > dir2/f3

git add .

git config user.name "John Doe & John Other"
git config user.email "johndoe@example.com, johnother@example.com"
git commit -m "Commit 2"

# Test merge-tag
git config user.name "John Doe"
git config user.email johndoe@example.com

git branch dev
git checkout dev

echo ";qjkxb" > dir/f4

git add dir/f4
git commit -m "Release"
git tag -a v1 -m "Some version"
git checkout master

## defer eyes.Open()
## eyes.Close()

git cat-file tag $(cat .git/refs/tags/v1) | head -n4 | sed 's/v1/v1sig/g' > sigobj
cat >>sigobj <<EOF

Some signed version
-----BEGIN PGP SIGNATURE-----
NotReallyABase64Signature
ButItsGoodEnough
-----END PGP SIGNATURE-----
EOF

cat <(printf "tag %d\0" $(wc -c sigobj | cut -d' ' -f1); cat sigobj) > sigtag
FILE=.git/objects/$(sha1sum sigtag | cut -d' ' -f1 | sed 's/../\0\//')
mkdir -p $(dirname ${FILE})
cat sigtag | zlib-flate -compress > ${FILE}
echo $(sha1sum sigtag | cut -d' ' -f1) > .git/refs/tags/v1sig

git merge v1sig --no-ff -m "Merge tag v1"

# Test encoding
git config i18n.commitencoding "ISO-8859-1"
echo "fgcrl" > f6
git add f6
git commit -m "Encoded"

# Test iplBlob/tree tags
git tag -a v1-file -m "Some file" 933b7583b7767b07ea4cf242c1be29162eb8bb85
git tag -a v1-tree -m "Some tree" 672ef117424f54b71e5e058d1184de6a07450d0e

# Create test 'signed' objects

git cat-file commit $(cat .git/refs/heads/master) | head -n4 > sigobj
echo "gpgsig -----BEGIN PGP SIGNATURE-----" >> sigobj
echo " " >> sigobj
echo " NotReallyABase64Signature" >> sigobj
echo " ButItsGoodEnough" >> sigobj
echo " -----END PGP SIGNATURE-----" >> sigobj
echo "" >> sigobj
echo "Encoded" >> sigobj

cat <(printf "commit %d\0" $(wc -c sigobj | cut -d' ' -f1); cat sigobj) > sigcommit
FILE=.git/objects/$(sha1sum sigcommit | cut -d' ' -f1 | sed 's/../\0\//')
mkdir -p $(dirname ${FILE})
cat sigcommit | zlib-flate -compress > ${FILE}

git cat-file commit $(cat .git/refs/heads/master) | head -n4 > sigobj
echo "gpgsig -----BEGIN PGP SIGNATURE-----" >> sigobj
echo " Version: 0.1.2" >> sigobj
echo " " >> sigobj
echo " NotReallyABase64Signature" >> sigobj
echo " ButItsGoodEnough" >> sigobj
echo " -----END PGP SIGNATURE-----" >> sigobj
echo " " >> sigobj
echo "" >> sigobj
echo "Encoded" >> sigobj

cat <(printf "commit %d\0" $(wc -c sigobj | cut -d' ' -f1); cat sigobj) > sigcommit
FILE=.git/objects/$(sha1sum sigcommit | cut -d' ' -f1 | sed 's/../\0\//')
mkdir -p $(dirname ${FILE})
cat sigcommit | zlib-flate -compress >> ${FILE}
rm sigobj sigcommit

# Create test archive, clean up

tar czf git.tar.gz .git
mv git.tar.gz ${CUR_DIR}/testdata.tar.gz
cd ${CUR_DIR}
rm -rf ${TEST_DIR}
