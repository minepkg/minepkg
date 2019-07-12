#!bin/sh

echo "installing minepkg"
mkdir -p $HOME/.minepkg
curl -L "https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64" -o $HOME/.minepkg/minepkg
chmod +x $HOME/.minepkg/minepkg
sudo ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg