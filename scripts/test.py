from subprocess import call
import time
from threading import Thread
def send_Requests(namespace, args):
    print "hi"
    for x in range(0,10):
        url= 'localhost:80/'+namespace+args+'/'+str(x)
        call(['sudo','docker','tag','ubuntu',url])
        call(['sudo', 'docker','push',url])
try:
    t = Thread(None,send_Requests,None,('n1'))
    t.start()
    t1 = Thread(None,send_Requests,None,('n2'))
    t1.start()
    t2 = Thread(None,send_Requests,None,('n3'))
    t2.start()
    t3 = Thread(None,send_Requests,None,('n4'))
    t3.start()
    t4 = Thread(None,send_Requests,None,('n5'))
    t4.start()
    t5 = Thread(None,send_Requests,None,('n6'))
    t5.start()

    #thread.start_new_thread( send_Requests, (n2))
    #thread.start_new_thread( send_Requests, (n3))
    #thread.start_new_thread( send_Requests, (n4))
   #thread.start_new_thread( send_Requests, (n5))
   #thread.start_new_thread( send_Requests, (n6))
except: 
    print "well then"

while 1:
    pass
