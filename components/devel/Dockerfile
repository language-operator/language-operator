FROM git.theryans.io/langop/base:latest

# Install development tools
RUN apk add --no-cache \
    git \
    make \
    bash \
    curl \
    wget \
    vim \
    nano \
    sudo \
    uv

# Give langop user sudo privileges
RUN echo "langop ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

# Set bash as default shell
ENV SHELL=/bin/bash

# Set working directory and ensure it's owned by langop user
WORKDIR /workspace
RUN chown -R langop:langop /workspace

# End as langop user - entrypoint will handle the switch at runtime
USER langop

# Default command
CMD ["/bin/bash"]
