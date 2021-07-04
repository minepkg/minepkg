FROM adoptopenjdk:16-jre-openj9

# add minepkg
ADD https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64 /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

RUN mkdir /etc/minepkg && echo 'useSystemJava=true\n' > /etc/minepkg/config.toml

CMD ["/usr/bin/minepkg"]
