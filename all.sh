cd ..

if [[ -z $1 ]]; then
echo "
*********************************************************
*           Defaulting... Checking out Master
*********************************************************"
branch=master
else
echo "
*********************************************************
*           Checking out the branch '" $1 "'
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
        echo -n "Updating: " $1 
        git checkout -q $2 > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo " ===>  now on " $2
            echo ""
        else 
             if [[ ! -z $2 ]]
             then
                git checkout -q master 
                echo " ===>  now on " master
                echo ""
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
checkout btcrpcclient $branch
checkout btcutil $branch
checkout btcws $branch
checkout factom $branch
checkout factom-cli  $branch
checkout gobundle  $branch
checkout goleveldb  $branch

checkout btcjson $branch
checkout btclog  $branch
checkout dynrsrc $branch
checkout ed25519 $branch
checkout fastsha256  $branch
checkout gocoding master
checkout go-flags  $branch
checkout go-socks  $branch
checkout seelog  $branch
checkout snappy-go  $branch
checkout websocket  $branch

echo "
************************************************************** 
*         Compiling fctwallet, the cli, and factomd
************************************************************** 
"
compile factoid/fctwallet 
compile factom-cli  
compile FactomCode/factomd 
echo ""
echo ""
cd FactomCode


   
