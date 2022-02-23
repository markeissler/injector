#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------

# To fully customize the contents of this image, use the following Dockerfile instead:
# https://github.com/microsoft/vscode-dev-containers/tree/v0.122.1/containers/javascript-node-10/.devcontainer/Dockerfile
FROM mcr.microsoft.com/vscode/devcontainers/typescript-node:0-16

ENV BUILD_HOME="/home/node"
ENV BUILD_USER='node'
ENV BUILD_LOCK="${BUILD_HOME}/.locks"
ENV SOURCE_DIR="${BUILD_HOME}/source"

# locally installed binaries
ENV PATH="${BUILD_HOME}/bin:${SOURCE_DIR}/node_modules/.bin:${PATH}"

RUN apt-get update \
    && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends direnv

# Update build user shell configuration
RUN set -x \
    && printf "\nexport TERM=xterm" >> "${BUILD_HOME}/.bash_profile" \
    && for f in .bash_profile .bashrc .profile; do printf "\neval \"\$(direnv hook bash)\"\n\n" >> "${BUILD_HOME}/${f}"; done

# Add some aliases
RUN set -x \
    && for f in .bash_profile .bashrc .profile; do printf "\n" >> "${BUILD_HOME}/${f}"; done \
    && for f in .bash_profile .bashrc .profile; do printf "\nalias git-lg0=\"git log --pretty=oneline --abbrev-commit\"" >> "${BUILD_HOME}/${f}"; done \
    && for f in .bash_profile .bashrc .profile; do printf "\nalias git-ready-dev=\"git log --oneline dev ^master\"" >> "${BUILD_HOME}/${f}"; done \
    && for f in .bash_profile .bashrc .profile; do printf "\nalias git-ready-develop=\"git log --oneline develop ^master\"" >> "${BUILD_HOME}/${f}"; done

# Add git bash completion
RUN set -x \
    && curl "https://raw.githubusercontent.com/git/git/master/contrib/completion/git-completion.bash" -o "${BUILD_HOME}/.git-completion.bash"

# Add some useful additions to bashrc
COPY "bash_additions.sh" "/tmp/"
RUN set -x \
    && for f in .bash_profile .bashrc .profile; do printf "\n" >> "${BUILD_HOME}/${f}"; done \
    && for f in .bash_profile .bashrc .profile; do cat "/tmp/bash_additions.sh" >> "${BUILD_HOME}/${f}"; done \
    && rm "/tmp/bash_additions.sh"

# Configure the main working directory. This is the base directory used in any
# further RUN, COPY, and ENTRYPOINT commands.
RUN mkdir "${BUILD_HOME}/bin" "${BUILD_LOCK}" "${SOURCE_DIR}" "${SOURCE_DIR}/node_modules"
WORKDIR "${BUILD_HOME}"

# Fix permissions on BUILD_HOME directory
RUN set -x \
    && chown -R "${BUILD_USER}":"${BUILD_USER}" "${BUILD_HOME}" "${BUILD_LOCK}" "${SOURCE_DIR}"

USER "${BUILD_USER}"
