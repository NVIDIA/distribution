from subprocess import call
def clearAndPull():
    arr =['a','b','c','d']
    for x in arr:
        call("./wipe.sh",shell=True)
        call(['docker','pull','localhost:80/'+x+'/170:testing'])
clearAndPull()
