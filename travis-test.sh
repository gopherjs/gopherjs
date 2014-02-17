#! /bin/bash
set -e

# sudo apt-get install rlwrap
# wget https://launchpad.net/~rwky/+archive/nodejs-unstable/+files/nodejs_0.11.11-rwky1~precise_amd64.deb
# sudo dpkg -i nodejs_0.11.11-rwky1~precise_amd64.deb

echo "get fresh nvm"
curl https://raw.github.com/creationix/nvm/master/install.sh | sh
echo "loading nvm"
. $HOME/.nvm/nvm.sh
echo "installing node"
nvm install 0.11
echo "using node"
nvm use 0.11

which node
node -v