#!bin/sh

echo "installing minepkg"
sudo curl -L "https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64" -o /usr/local/bin/minepkg && sudo chmod +x /usr/local/bin/minepkg && sudo chown $USER /usr/local/bin/minepkg
