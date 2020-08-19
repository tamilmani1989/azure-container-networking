#!/bin/bash
set -e

# Save current working directory
PWD=`pwd`
pushd $PWD

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $DIR/src/DncRP && pwd

echo '---------- Install asp.net package ----------'

sudo apt-get update; \
  sudo apt-get install -y apt-transport-https && \
  sudo apt-get update && \
  sudo apt-get install -y dotnet-sdk-3.1

Operation=$1
BuildConfig=$2
Runtime=$3

#!/bin/bash
## declare an array variable
declare -a parray=("Server/DncRP.csproj"
                   "Tests/UnitTests/DncRP.Unittests.csproj")

# get length of an array
arraylength=${#parray[@]}

case "$Operation" in
    "build" )

	# loop through each file
	for (( i=0; i<${arraylength}; i++ ));
	do
		selfContained=false
		pfile=${parray[$i]}
		pdir=$(dirname "$pfile")
		mkdir -p "$pdir/$Runtime"

		if [ "$pfile" = "Server/DncRP.csproj" ]; then
			selfContained=true
		fi

		echo "---------- Publish $pfile ----------"
		dotnet publish "$pfile" --configuration $BuildConfig -r $Runtime -o "$pdir/$Runtime" --self-contained $selfContained; EX=$?

		if [ "$EX" -ne "0" ]; then
			popd
			echo "Failed to build $pfile"
			exit $EX
		fi
	done
	;;
	"test" )

	echo "---------- Running unit tests ----------"
	dotnet test Tests/UnitTests/DncRP.Unittests.csproj --configuration $BuildConfig --runtime $Runtime --collect:"XPlat Code Coverage" -r ./codecoverage; EX=$?

	if [ "$EX" -ne "0" ]; then
		popd
		echo "DncRP unit test failed"
		exit $EX
    fi
	;;
esac


# Restore working directory
popd

# Exit with explicit 0 exit code to make sure build passes
exit $EX