cd ..

if [[ -z $1 ]]; then
echo "
*********************************************************
*       Defaulting... Checking out Master
*********************************************************"
branch=master
else
echo "
*********************************************************
*       Checking out the" $1 "branch
*********************************************************"
branch=$1
fi
# We cd to the given directory, look and see if the branch exists...
# If it does, we make sure we are in that branch.
# Then we go back to the previous directory.
#
# In all cases we do a pull.  Something might have changed.
checkout() {
    current=`pwd` 
    cd $1
    if [ $? -eq 0 ]; then
        echo $1 | awk "{printf(\"%15s\",\"$1\")}"
        git checkout -q $2 > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo -e " now on" $2    # checkout did not fail
        else 
            git checkout -q master > /dev/null 2>&1
            if [ $? -ne 0 ]; then
                echo -e " ****checkout failed!!!"
            else
                echo " defaulting to master"
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

checkout FactomCode $branch
checkout btcd $branch
checkout factoid $branch
checkout factom $branch
checkout factom-cli  $branch
checkout btcrpcclient $branch
checkout btcutil $branch
checkout btcws $branch
checkout gobundle  $branch
checkout goleveldb  $branch

checkout gocoding master

checkout btcjson $branch
checkout btclog  $branch
checkout dynrsrc $branch
checkout ed25519 $branch
checkout fastsha256  $branch
checkout go-flags  $branch
checkout go-socks  $branch
checkout seelog  $branch
checkout snappy-go  $branch
checkout websocket  $branch

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


   
