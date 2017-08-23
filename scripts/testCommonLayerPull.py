from subprocess import call
def clearAndPull():
    arr =['a','b','c','d']
    for x in arr:
        for y in range(180,185):
            call(['docker','pull','localhost:80/'+x+'/'+str(y)+':testing'])
            call("./wipe.sh",shell=True)
clearAndPull()
