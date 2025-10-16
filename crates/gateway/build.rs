fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../../proto");

    let proto_files = [
        "../../proto/smartticket/ticket.proto",
        "../../proto/smartticket/knowledge.proto",
        "../../proto/smartticket/user.proto",
        "../../proto/smartticket/tenant.proto",
        "../../proto/smartticket/role_permission.proto",
        "../../proto/smartticket/sla.proto",
        "../../proto/smartticket/common.proto",
    ];

    let proto_includes = ["../../proto"];

    // Generate tonic code
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .out_dir("src/proto")
        .compile(&proto_files, &proto_includes)?;

    Ok(())
}
