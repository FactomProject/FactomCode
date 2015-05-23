cd ..
echo checking out $1

# We cd to the given directory, look and see if the branch exists...
# If it does, we make sure we are in that branch.
# Then we go back to the previous directory.
#
# In all cases we do a pull.  Something might have changed.
checkout() {
    cd $1
    echo $1
    git pull
    if [[ `git branch --list $2` ]]
    then
        git checkout $2
    fi
    cd ..
}

checkout btcd $1
checkout btcjson $1
checkout btclog  $1
checkout btcrpcclient $1
checkout btcutil $1
checkout btcwallet $1
checkout btcws  $1
checkout dynrsrc $1
checkout factom $1
checkout factom-cli  $1
checkout FactomDocs $1
checkout factomexplorer  $1
checkout fastsha256  $1
checkout gobundle  $1
cd gocoding 
git pull
cd ..
checkout go-flags  $1
checkout goleveldb  $1
checkout go-socks  $1
checkout seelog  $1
checkout snappy-go  $1
checkout website  $1
checkout websocket  $1
checkout WorkItems $1
checkout FactomCode $1

cd FactomCode

