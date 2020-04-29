FROM adoptopenjdk:8-jre-hotspot

# add minepkg
ADD https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64 /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

RUN echo 'useSystemJava=true\n' > /etc/minepkg/config.toml

CMD ["/usr/bin/minepkg"]
