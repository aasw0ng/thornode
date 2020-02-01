#!/bin/sh

source $(dirname "$0")/core.sh

PEER="${PEER:=none}" # the hostname of a seed node
SIGNER_NAME="${SIGNER_NAME:=thorchain}"
SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

if [ ! -f ~/.thord/config/genesis.json ]; then
    if [[ "$PEER" == "none" ]]; then
        echo "Missing PEER"
        exit 1
    fi

    thorcli keys show $SIGNER_NAME || echo $SIGNER_PASSWD | thorcli --trace keys add $SIGNER_NAME 2>&1

    # write private key to tss volume
    if [ ! -z ${TSSPRIVKEY+x} ]; then
        echo $SIGNER_PASSWD | thorcli keys tss $SIGNER_NAME 2> $TSSPRIVKEY
    fi

    NODE_ADDRESS=$(thorcli keys show $SIGNER_NAME -a)
    init_chain $NODE_ADDRESS

    fetch_genesis $PEER

    NODE_ID=$(fetch_node_id $PEER)
    peer_list $NODE_ID $PEER

    if [[ "$NET" == "mocknet" ]]; then
        # create a binance wallet and bond/register
        gen_bnb_address
        ADDRESS=$(cat ~/.bond/address.txt)

        # send bond transaction to mock binance
        $(dirname "$0")/mock-bond.sh $PEER $ADDRESS $NODE_ADDRESS

        sleep 15 # wait for thorchain to register the new node account

        # set node keys
        echo $SIGNER_PASSWD | thorcli tx thorchain set-node-keys $(thorcli keys show thorchain --pubkey) $(thorcli keys show thorchain --pubkey) $(thord tendermint show-validator) --node tcp://$PEER:26657 --from $SIGNER_NAME --yes
    elif [[ "$NET" == "testnet" ]]; then
        # create a binance wallet
        gen_bnb_address
        ADDRESS=$(cat ~/.bond/address.txt)

        # wait for bond transaction and for node account to be registered
        BOND_FILE=/tmp/bonded
        while [ ! -f $BOND_KEY ]; do
            sleep 10
        done
        # set node keys
        echo $SIGNER_PASSWD | thorcli tx thorchain set-node-keys $(thorcli keys show thorchain --pubkey) $(thorcli keys show thorchain --pubkey) $(thord tendermint show-validator) --node tcp://$PEER:26657 --from $SIGNER_NAME --yes

    else
        echo "YOUR NODE ADDRESS: $NODE_ADDRESS . Send your bond with this as your address."
    fi

fi

exec "$@"
