#!/bin/bash

set -e 
set -o pipefail

echo "installing minepkg"
mkdir -p $HOME/.minepkg

bin="minepkg-linux-amd64"

if [[ "$OSTYPE" == "darwin"* ]]; then
  echo "installing macOS binary"
  bin="minepkg-macos-amd64"
elif [[ "$OSTYPE" == "cygwin" ]]; then
  echo "Installation with cygwin is not supported. Please install throug powershell."
  echo "You should still be able to use minepkg with cygwin after that."
  exit 1
elif [[ "$OSTYPE" == "msys" ]]; then
  echo "Installation with MinGW is not supported. Please install throug powershell."
  echo "You should still be able to use minepkg with MinGW after that."
  exit 1
elif [[ "$OSTYPE" == "freebsd"* ]]; then
  echo "Installation on freebsd currently is not supported. Ping us if you want this."
  echo "https://github.com/fiws/minepkg"
  exit 1
fi

curl -L "https://storage.googleapis.com/minepkg-client/latest/$bin" -o $HOME/.minepkg/minepkg
chmod +x $HOME/.minepkg/minepkg
if [ "$(id -u)" -eq 0 ]; then
  ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg
else
  echo ""
  echo "Attempting to symlink the binary to your system"
  echo "You can abort and enter this command yourself if you prefer:"
  echo "  sudo ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg"
  sudo ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg
fi

echo ""
echo "minepkg should now be installed!"
minepkg --version