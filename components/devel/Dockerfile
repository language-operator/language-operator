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

# Give based user sudo privileges
RUN echo "based ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

# Set bash as default shell
ENV SHELL=/bin/bash

# Set working directory and ensure it's owned by based user
WORKDIR /workspace
RUN chown -R based:based /workspace

# End as based user - entrypoint will handle the switch at runtime
USER based

# Default command
CMD ["/bin/bash"]
