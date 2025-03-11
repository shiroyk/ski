#!/bin/sh -e
# this is just the commit it was last tested with
sha=fa4477ae5ce31b672238ea37759642369b4eaec2

mkdir -p wpt
cd wpt
git init
git remote add origin https://github.com/web-platform-tests/wpt
git sparse-checkout init --cone
git sparse-checkout set resources common fetch url FileAPI encoding
git fetch origin --depth=1 "${sha}"
git reset --hard "${sha}"
cd -