import os
import sys

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Expected 1 argument, {} were given".format(len(sys.argv) - 1))
    file = sys.argv[1]
    with open(file, 'r') as txs:
        for line in txs:
            recipient, amount = line.split()
            os.system('./bin/noobcash-cli t {} {}'.format(recipient, amount))
    file.close()