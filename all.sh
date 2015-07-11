# Copyright 2015 Factom Foundation
# Use of this source code is governed by the MIT
# license that can be found in the LICENSE file.

# To be run from within the FactomCode project.
#
# Factom has a pile of dependencies, and development requries that these be 
# kept in sync with each other.  This script allows you to check out a 
# particular branch in many repositories, while specifying a default branch.
#
# So for example, if you want to check development, and default to master:
#  
#  ./all.sh development master
#
# Or if you have your own branch TerribleBug, building off development:
#
#  ./all.sh TerribleBug development
#
# Any repository that doesn't have a development branch in this last case
# is going to default to master.
#
cd ..

if [[ -z $1 ]]; then
echo "
*********************************************************
*       Defaulting... Checking out Master
*
*       ./all.sh <branch> <default>
*
*       Will try to check out <branch>, will default
*       to <default>, and if neither exists, will 
*       checkout the master branch.
*
*********************************************************"
branch=master
default=
else
echo "
*********************************************************
*       Checking out the" $1 "branch
*
*       ./all.sh <branch> <default>
*
*       Will try to check out <branch>, will default
*       to <default>, and if neither exists, will 
*       checkout the master branch.
*
*********************************************************"
branch=$1
if [[ -z $2 ]]; then
default=master
else
default=$2
fi
fi
checkout() {
    current=`pwd` 
    cd $1
    if [ $? -eq 0 ]; then
        echo $1 | awk "{printf(\"%15s\",\"$1\")}"
        git checkout -q $2 > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo -e " now on" $2    # checkout did not fail
        else 
            git checkout -q $3 > /dev/null 2>&1
            if [ $? -ne 0 ]; then
                git checkout -q master > /dev/null 2>&1
                if [ $? -ne 0 ]; then
                   echo -e " ****checkout failed!!!"
                else
                   echo " defaulting to master"
                fi
            else
                echo " defaulting to" $3
            fi
        fi
        git pull | awk '$1!="Already" {print}'
        cd $current
   fi
}

compile() {
    current=`pwd`
    cd $1
    echo "Compiling: " $1
    go clean
    go install
    cd $current
}

checkout FactomCode   $branch $default
checkout btcd         $branch $default
checkout factoid      $branch $default
checkout factom       $branch $default
checkout factom-cli   $branch $default
checkout btcrpcclient $branch $default
checkout btcutil      $branch $default 
checkout btcws        $branch $default
checkout gobundle     $branch $default
checkout goleveldb    $branch $default

checkout gocoding     master  $default

checkout btcjson      $branch $default
checkout btclog       $branch $default
checkout dynrsrc      $branch $default
checkout ed25519      $branch $default
checkout fastsha256   $branch $default 
checkout go-flags     $branch $default
checkout go-socks     $branch $default
checkout seelog       $branch $default
checkout snappy-go    $branch $default
checkout websocket    $branch $default

echo "
******************************************************** 
*     Compiling fctwallet, the cli, and factomd
******************************************************** 
"
compile factoid/fctwallet 
compile factom-cli  
compile FactomCode/factomd 
echo ""
echo ""
cd FactomCode
