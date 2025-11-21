FROM scratch
COPY sshdtui /usr/bin/sshdtui
ENV HOME=/home/user
ENTRYPOINT ["/usr/bin/sshdtui"]
