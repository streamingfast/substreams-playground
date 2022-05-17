#[macro_export]
macro_rules! field {
    ($a:expr, $b:expr, $c:expr) => {
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

#[macro_export]
macro_rules! field_from_strings {
    ($name:expr, $value:expr) => {
        Field {
            name: $name.to_string(),
            new_value: String::from_utf8_lossy($value.new_value.as_slice()).to_string(),
            old_value: String::from_utf8_lossy($value.old_value.as_slice()).to_string(),
        }
    };
}
