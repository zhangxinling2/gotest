local val = redis.call('get',KEYS[1])
if val==false then
    --可以进行加锁
    return redis.call('set',KEYS[1],ARGV[1],'EX', ARGV[2])
elseif val==ARGV[1] then
    redis.call('expire',KEYS[1],ARGV[2])
    --因为set的返回值也是ok
    return 'OK'
else
    --锁被别人拿着
    return ''
end