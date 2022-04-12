#[macro_export]
macro_rules! field {
    ($a:expr, $b:expr, $c:expr) => {
        //todo: send delta object entirely instead of sending old and new value
        // as seperate arguments and add casting
        Field {
            name: $a.to_string(),
            new_value: $b.to_string(),
            old_value: $c.to_string(),
        }
    };
}

#[macro_export]
macro_rules! field_create_string {
    ($name:expr, $value:expr) => {
        Field {
            name: $name.to_string(),
            new_value: $value.to_string(),
            old_value: "".to_string(),
        }
    };
}

//fixme: probably gonna remove this macro as it doesn't really seem good
#[macro_export]
macro_rules! proto_decode_to_string {
    ($a:expr, $b:expr) => {
        if $a.len() == 0 {
            $b.to_string()
        } else {
            proto::decode($a).unwrap()
        }
    };
}
