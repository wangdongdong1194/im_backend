#!/bin/bash

# 压缩当前目录
# 命名：im_backend.zip
# 排除：.git 目录
zip -r im_backend.zip . -x "*/.git/*"

# 提示完成
echo "✅ 压缩完成！生成文件：im_backend.zip"