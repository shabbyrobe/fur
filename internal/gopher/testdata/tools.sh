#!/usr/bin/env bash
script_abspath="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

sels=(
    "1/world/search-hosts%3Fcom%20net%20info%20land%20org"
    "1/world/search-hosts%3Fedu%20gov%20mil"
    "1/world/search-hosts%3Fca%20us%20gp"
    "1/world/search-hosts%3Far%20co%20mx%20pe%20br"
    "1/world/search-hosts%3Fat%20be%20ch%20cr%20cz%20de%20dk%20ee%20eu%20fi%20fr%20gr%20hr%20hu%20is%20it%20lt%20me%20nl%20es%20no%20pl%20pt%20ro%20ru%20se%20su%20ua%20uk"
    "1/world/search-hosts%3Fau%20cn%20hk%20in%20io%20jp%20nz%20pw%20tk%20th%20tm%20tw%20cx%20za"
    "1/world/search-hosts%3Fblack%20club%20life%20moe%20name%20ninja%20online%20productions%20solutions%20space%20technology%20tips%20town%20works%20zone%20party%20engineering"
)

dl-save() {
    url="$1"
    fname="$2"

    ok=""
    for (( i=0; i<3; i++ )); do
        if out="$( fur --txt "$url" )"; then
            ok=1
            break
        fi
        echo "retry $i"
        sleep 1
    done

    if [[ -z "$ok" ]]; then
        return 1
    fi

    echo -en "FUR-DUMP $( date -Is ) $url\n$out" > "$fname"
}

cmd-hosts() {
    for sel in "${sels[@]}"; do
        fur -x=i -j gopher://gopher.floodgap.com/"$sel" |
            jq -r '.hostname'
    done
}

cmd-caps-all() {
    cmd-hosts | while read -r host; do
        if [[ ! -f "caps-$host.txt" ]]; then
            echo "$host"
            url="gopher://$host/0caps.txt"
            dl-save "$url" "caps-$host.txt" || true
        fi
    done
}

cmd-robots-all() {
    cmd-hosts | while read -r host; do
        echo "$host"
        url="gopher://$host/0robots.txt"
        dl-save "$url" "robots-$host.txt" || true
    done
}

cmd-dump() {
    kind="$1"
    pushd "$script_abspath" > /dev/null
        find . -type f -name "$kind*.txt" | sort | while read -r line; do
            echo "-----------------------------------------"
            echo "$kind: $line"
            echo "-----------------------------------------"
            cat "$line"
            echo 
        done
    popd > /dev/null
}

"cmd-$1" "${@:2}"
