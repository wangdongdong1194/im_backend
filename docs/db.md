# 数据库操作

- 根据erp查询好友
    ``` mysql
    SELECT u.*
    FROM users self
    JOIN friendships f ON self.id = f.user_id
    JOIN users u ON f.friend_id = u.id
    WHERE self.erp = 123;
    ```