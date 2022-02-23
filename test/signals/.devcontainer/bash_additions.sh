function parse_git_dirty {
  _str=$(git status --porcelain --untracked 2> /dev/null | tail -n1);

  if [[ "${_str}" == "??"* ]]; then
    echo "+";
  elif [[ "${_str}" == " D "* ]]; then
    echo "-"
  elif [[ "${_str}" == " M "* ]]; then
    echo "*"
  fi
}
function parse_git_branch {
  git branch --no-color 2> /dev/null | sed -e '/^[^*]/d' -e "s/* \(.*\)/[\1$(parse_git_dirty)]/"
}
export PS1='\n\u@\h::\[\033[1;33m\]\w\[\033[0m\]$(parse_git_branch)\nprompt> '

#
# Install git bash completion...
#
# curl https://raw.githubusercontent.com/git/git/master/contrib/completion/git-completion.bash -o ~/.git-completion.bash
#

# Load git bash completion
[[ -s "$HOME/.git-completion.bash" ]] && source "$HOME/.git-completion.bash"
