from multiprocessing.dummy import Pool as ThreadPool
from subprocess import call
import time

def send_Requests(namespace):
    print "hi"
    for x in range(180,185):
        url= 'localhost:80/'+namespace+'/'+str(x)+':testing'
        call(['docker','tag','alpine',url])
        call(['docker','push',url])

pool = ThreadPool(4)
arr=['a','b','c','d']
results=pool.map(send_Requests,arr)

pool.close()
pool.join()
