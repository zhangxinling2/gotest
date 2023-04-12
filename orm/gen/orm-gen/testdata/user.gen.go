package testdata

import(
    "gotest/orm"
    
    "database/sql"
    
)

//希望通过外面传进去



const(
    //拼接你的类型名字，你的字段名字
    UserName = "Name"
    //拼接你的类型名字，你的字段名字
    UserAge = "Age"
    //拼接你的类型名字，你的字段名字
    UserNickName = "NickName"
    //拼接你的类型名字，你的字段名字
    UserPicture = "Picture"
)
        func UserNameLt(val string) orm.Predicate{
        return orm.C("Name").Lt(val)
        }
        func UserNameGt(val string) orm.Predicate{
        return orm.C("Name").Gt(val)
        }
        func UserAgeLt(val *int) orm.Predicate{
        return orm.C("Age").Lt(val)
        }
        func UserAgeGt(val *int) orm.Predicate{
        return orm.C("Age").Gt(val)
        }
        func UserNickNameLt(val *sql.NullString) orm.Predicate{
        return orm.C("NickName").Lt(val)
        }
        func UserNickNameGt(val *sql.NullString) orm.Predicate{
        return orm.C("NickName").Gt(val)
        }
        func UserPictureLt(val []byte) orm.Predicate{
        return orm.C("Picture").Lt(val)
        }
        func UserPictureGt(val []byte) orm.Predicate{
        return orm.C("Picture").Gt(val)
        }

const(
    //拼接你的类型名字，你的字段名字
    UserDetailAddress = "Address"
)
        func UserDetailAddressLt(val string) orm.Predicate{
        return orm.C("Address").Lt(val)
        }
        func UserDetailAddressGt(val string) orm.Predicate{
        return orm.C("Address").Gt(val)
        }


