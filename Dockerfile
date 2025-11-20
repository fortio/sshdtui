FROM scratch
COPY sshd /usr/bin/sshd
ENV HOME=/home/user
ENTRYPOINT ["/usr/bin/sshd"]
