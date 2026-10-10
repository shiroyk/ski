#!/bin/sh -e
# this is just the commit it was last tested with, the v26.8.1 tag
sha=7be6d3af31a65adea57c94c41e50c2b071ed0b3a

mkdir -p node
cd node
git init
git remote add origin https://github.com/nodejs/node
git sparse-checkout init --cone
git sparse-checkout set test/parallel test/common
git fetch origin --depth=1 "${sha}"
git reset --hard "${sha}"
cd -
