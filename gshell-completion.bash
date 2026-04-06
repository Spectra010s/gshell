#!/bin/bash

_gshell_completion() {
    local cur prev subcmd
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    subcmd="${COMP_WORDS[1]}"

    local subcommands="send get sync run open status help"

    # First argument - complete subcommands and help flags
    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$subcommands --help -h" -- "$cur"))
        return 0
    fi

    # Check if -r is already in the command words
    local has_r=0
    for word in "${COMP_WORDS[@]}"; do
        if [ "$word" = "-r" ]; then has_r=1; break; fi
    done

    case "$subcmd" in
        "send"|"get"|"sync")
            if [[ "$cur" == -* ]]; then
                # build flags list
                local flags=""
                if [ "$has_r" -eq 0 ]; then flags="$flags -r"; fi
                if [ "$subcmd" = "send" ] || [ "$subcmd" = "get" ]; then
                    flags="$flags -n --dry-run"
                fi
                COMPREPLY=($(compgen -W "$flags" -- "$cur"))
            else
                COMPREPLY=($(compgen -f -- "$cur"))
            fi
            ;;
        "run")
            COMPREPLY=()
            ;;
        *)
            COMPREPLY=()
            ;;
    esac

    return 0
}

complete -F _gshell_completion gshell

