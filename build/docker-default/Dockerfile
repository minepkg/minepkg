FROM adoptopenjdk:8-jre-hotspot

# add minepkg
ADD https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64 /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

# TODO: set minepkg to always use system jdk!

CMD ["/usr/bin/minepkg"]
