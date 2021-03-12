#[test]
fn test_wrong() {
    assert_eq!(1, 2);
}

#[test]
fn test_equal_strings() {
    assert_eq!("hello", "hello");
}

#[test]
fn test_unwrap_none() {
    let x: Option<&str> = None;
    assert!(x.unwrap() == "");
}

#[test]
fn test_unwrap_some() {
    let x: Option<&str> = Some("string");
    assert!(x.unwrap() == "string");
}


#[test]
fn test_equals() {
    assert!(2 == 1);
}
